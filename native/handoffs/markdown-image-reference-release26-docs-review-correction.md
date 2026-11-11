# .26 docs review: narrow corrections

ACTIVE2026-11-10, same GLM docs owner. Root has read final docs/result; two
corrections remain before acceptance. No product tests/builds/launches.

1. native/VALIDATION.md .26 says aggregate attempt1 was12/13. That verdict
   never existed: actual stage stderr says missing HostSHA256 and it stopped
   before writing attempt1 validator/aggregate. State original metadata attempt
   exit1 before verdict; corrected attempt2 PASS13/13. Check no other docs repeat
   the unsupported12/13 count.
2. The task required keeping the complete accepted .25/.24 validation sections.
   Current .25 section was rewritten, replacing its original identity table,
   dropping exact pre-close and close-request timestamps, and changing prose.
   Restore the accepted .25 section from your pre-edit original, changing only
   current heading/entry labels to historical. The accepted original was
   native/VALIDATION.md156787bytes SHA256
   53654C03EA77AC1D40EB91566B325BD85F68513536E8FECA37CB6682F5A34F10.
   Preserve .24 section as well; do not rewrite history to shorten it.

Concrete original .25 excerpt read by root before your edits:
```
最终validator为13/13通过，attempt1无重试或元数据修正。四个引擎manifest、
385个构建输入匹配，14个便携副本核对。一次打包调用生成五个GUI可执行
文件，全部exit0。

一次隐藏精确复制的宿主使用全新隔离配置，按顺序打开UTF-8+BOM/CRLF
自定义格式文本和PNG；在它唯一拥有的`WM_CLOSE`之前，已发布675字节严格
有序混合会话，活动索引1，会话SHA256
`21A967D39B5F1D2A0C530C776D30782969ED72EE58A507EA588D8F60AE2A3E2C`。
捕获时间为2026-11-10T07:37:33.5751458Z，关闭请求时间为
2026-11-10T07:37:33.5804387Z。正常关闭以更新的写入时间发布同一字节，
exit0，Job total1/active0，无强制清理、owned handles已释放，进程流
为空，且无worker请求。
```

Prefer existing pre-edit bytes or your preceding turn tool input over synthesis.
If original cannot be retrieved, report that concrete limitation and restore
all known factual content; do not falsely claim byte-identical section restore.
Keep corrected result handoff honest and publish refreshed six doc identities.
Accepted .26 package/evidence stay immutable; no further helper work.
