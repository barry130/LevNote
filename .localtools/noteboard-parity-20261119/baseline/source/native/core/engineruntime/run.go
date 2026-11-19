package engineruntime

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"sync"

	"levnote.local/native/core/enginecatalog"
	"levnote.local/native/core/engineipc"
)

func (r *Runner) run(ctx context.Context, cancel context.CancelCauseFunc, id engineipc.EngineID, src Source, selection selectedPackage) (result Result, err error, cleanupErr error) {
	descriptor, admissionErr := admittedEngine(id)
	if admissionErr != nil {
		return result, admissionErr, nil
	}
	if src.Size < 0 || src.Size > descriptor.SourceMaxBytes {
		return result, failure(FailureSourceLimit, "源码超过导出字节上限", nil), nil
	}
	if src.Data == nil {
		return result, failure(FailureSourceSize, "缺少不可变源码快照", nil), nil
	}
	if err = contextFailure(ctx); err != nil {
		return
	}
	prefix, prefixErr := engineipc.EncodeRequestPrefix(engineipc.RequestMeta{Engine: id, Format: descriptor.Format, Page: src.Page}, uint64(src.Size))
	if prefixErr != nil {
		return result, failure(FailureProtocol, "无法编码导出请求", prefixErr), nil
	}
	bundle, verifyErr := verifySelectedBundle(ctx, selection)
	if verifyErr != nil {
		var failed *Failure
		if errors.As(verifyErr, &failed) && failed.Kind == FailureCleanup {
			cleanupErr = verifyErr
		}
		return result, verifyErr, cleanupErr
	}
	result.Engine = bundle.info
	defer func() { cleanupErr = errors.Join(cleanupErr, bundle.close()) }()
	if r.hooks != nil && r.hooks.afterVerify != nil {
		r.hooks.afterVerify(bundle)
	}
	if err = contextFailure(ctx); err != nil {
		return
	}
	job, jobErr := newProcessJob()
	if jobErr != nil {
		return result, failure(FailureUnavailable, "无法建立导出进程生命周期", jobErr), nil
	}
	defer func() { cleanupErr = errors.Join(cleanupErr, job.closeAndWait()) }()
	args := []string{"--stdio-v1"}
	if r.hooks != nil {
		args = append(args, r.hooks.workerArgs...)
	}
	cmd := exec.CommandContext(ctx, bundle.exe, args...)
	cmd.Dir = bundle.dir
	cmd.WaitDelay = r.waitDelay
	configureCommand(cmd)
	stdin, pipeErr := cmd.StdinPipe()
	if pipeErr != nil {
		return result, failure(FailureUnavailable, "无法建立导出输入管道", pipeErr), nil
	}
	var closeInputOnce sync.Once
	closeInput := func() { closeInputOnce.Do(func() { _ = stdin.Close() }) }
	defer closeInput()
	decoder := engineipc.NewResponseDecoder(id, bundle.info.Version, uint64(src.Size))
	stdout := &responseSink{decoder: decoder, cancel: cancel}
	if r.hooks != nil {
		stdout.observe = r.hooks.observeStdout
	}
	stderr := &stderrSink{cancel: cancel}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Cancel = func() error {
		// Always address the precise root as well, covering the Start/attach gap.
		jobErr := job.terminate()
		killErr := cmd.Process.Kill()
		closeInput()
		if jobErr == nil || killErr == nil || errors.Is(killErr, os.ErrProcessDone) {
			return nil
		}
		return errors.Join(jobErr, killErr)
	}
	if startErr := cmd.Start(); startErr != nil {
		return result, failure(FailureUnavailable, "无法启动已验证的引擎程序", startErr), nil
	}
	if r.hooks != nil && r.hooks.afterStart != nil {
		r.hooks.afterStart(cmd.Process)
	}
	var attachErr error
	if r.hooks != nil && r.hooks.beforeAttach != nil {
		attachErr = r.hooks.beforeAttach()
	}
	if attachErr == nil {
		attachErr = job.attach(cmd.Process)
	}
	if attachErr != nil {
		killErr := cmd.Process.Kill()
		closeInput()
		waitErr := cmd.Wait() // Every successful Start is waited, even failed attachment.
		if cmd.ProcessState == nil {
			cleanupErr = errors.Join(killErr, waitErr)
		}
		if r.hooks != nil && r.hooks.afterWait != nil {
			r.hooks.afterWait()
		}
		return result, failure(FailureUnavailable, "无法把导出进程加入生命周期管理", attachErr), cleanupErr
	}
	if r.hooks != nil && r.hooks.afterAttach != nil {
		r.hooks.afterAttach(job)
	}
	writerDone := make(chan error, 1)
	go func() {
		writeErr := writeRequest(stdin, prefix, src)
		if writeErr != nil {
			cancel(writeErr)
		}
		closeInput()
		writerDone <- writeErr
	}()
	waitErr := cmd.Wait()
	if cmd.ProcessState == nil {
		cleanupErr = waitErr
	}
	if r.hooks != nil && r.hooks.afterWait != nil {
		r.hooks.afterWait()
	}
	// Wait closes its child pipes. Explicit closure releases the source writer
	// even when an early worker exit did not consume the complete request.
	closeInput()
	writerErr := <-writerDone
	if err = contextFailure(ctx); err != nil {
		return
	}
	if writerErr != nil {
		return result, writerErr, cleanupErr
	}
	if stdout.err != nil {
		return result, stdout.err, cleanupErr
	}
	if stderr.err != nil {
		return result, stderr.err, cleanupErr
	}
	if waitErr != nil {
		return result, failure(FailureWorkerExit, "导出程序异常退出或输出未完整关闭", waitErr), cleanupErr
	}
	if stderr.bytes != 0 {
		return result, failure(FailureUnexpectedStderr, "导出程序产生了意外诊断输出", nil), cleanupErr
	}
	meta, payload, decodeErr := decoder.Finish()
	if decodeErr != nil {
		return result, protocolFailure(decodeErr), cleanupErr
	}
	result.Warnings = meta.Warnings
	if meta.Status == "error" {
		engineErr := failure(FailureEngine, meta.Error.Message, nil)
		engineErr.Code = meta.Error.Code
		err = engineErr
		return
	}
	result.Payload = payload
	return
}

func admittedEngine(id engineipc.EngineID) (enginecatalog.Descriptor, error) {
	descriptor, ok := enginecatalog.Lookup(id)
	if !ok {
		return enginecatalog.Descriptor{}, failure(FailureUnavailable, "未知的内置导出引擎", nil)
	}
	return admittedPlatform(descriptor)
}

func protocolFailure(err error) error {
	if errors.Is(err, engineipc.ErrStdoutLimit) {
		return failure(FailureStdoutLimit, "导出程序输出超过上限", err)
	}
	return failure(FailureProtocol, "导出程序返回无效或不完整的数据", err)
}

// These sinks never block inside Write. os/exec Wait joins the copy goroutines
// before the coordinator reads their fields.
type responseSink struct {
	decoder *engineipc.ResponseDecoder
	cancel  context.CancelCauseFunc
	err     error
	observe func([]byte)
}

func (s *responseSink) Write(p []byte) (int, error) {
	n, err := s.decoder.Write(p)
	if err != nil {
		s.err = protocolFailure(err)
		s.cancel(s.err)
		return n, s.err
	}
	if s.observe != nil {
		s.observe(p[:n])
	}
	return n, nil
}

type stderrSink struct {
	bytes  int
	cancel context.CancelCauseFunc
	err    error
}

func (s *stderrSink) Write(p []byte) (int, error) {
	if len(p) > maxStderrBytes-s.bytes {
		s.err = failure(FailureStderrLimit, "导出程序诊断超过上限", nil)
		s.cancel(s.err)
		return 0, s.err
	}
	s.bytes += len(p) // Raw stderr is deliberately not retained or displayed.
	return len(p), nil
}

type exactWriter struct {
	dst           io.Writer
	size, written int64
	err           error
}

func (w *exactWriter) Write(p []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	if int64(len(p)) > w.size-w.written {
		w.err = failure(FailureSourceSize, "源码快照写出字节超过声明长度", nil)
		return 0, w.err
	}
	n, err := w.dst.Write(p)
	w.written += int64(n)
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	if err != nil {
		w.err = failure(FailureSourceSize, "无法完整发送源码快照", err)
	}
	return n, w.err
}

func writeRequest(dst io.Writer, prefix []byte, src Source) error {
	if n, err := dst.Write(prefix); err != nil || n != len(prefix) {
		return failure(FailureSourceSize, "无法发送导出请求头", err)
	}
	w := &exactWriter{dst: dst, size: src.Size}
	reported, err := src.Data.WriteTo(w)
	if w.err != nil {
		return w.err
	}
	if err != nil || reported != w.written || w.written != src.Size {
		return failure(FailureSourceSize, "源码快照实际字节数与声明不符", err)
	}
	return nil
}
