# Restore route checkpoint adjudication

2026-11-10 root read the GLM partial review and actual Restore/binaryLane
implementation. Restore is synchronous on the UI owner, waits on task.done
without pumping UI callbacks, and the worker owns only binaryRequest/output.
Its current comment explicitly prohibits nested UI adoption. No source-grounded
capacity-changing event between the loop's early check and appendBinary has
been demonstrated. Do not classify the missing duplicate check alone as a
reachable current bug or make a test mutate App from the worker goroutine.

A final restore admission check is harmless defensive consistency if the owner
chooses it; it is not a required new concurrency mechanism, test hook or scope
expansion. Existing lane take/release and local out lifetime already drop a
refused output; don't invent disposable resources in BinaryFile metadata.
Focus final review on reachable asynchronous open/Flush callback paths and
successful100-record mixed recovery. The genuine async binary path already
needs its independent post-IO/post-Flush checks, as the original task says.
