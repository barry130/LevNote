# 格式与互操作 样本

这份文件用来验证 doc-block 的往返：读进来再写回去，未改动的块应逐字节不变。

```doc-block
id: arch
kind: diagram
mode: reference
src: diagrams/arch.drawio
title: 整体架构
---
```

中间夹一段正文。

````doc-block
id: snippet-py
kind: code
mode: inline
---
```python
print("hello")
```
````

末尾再来一个思维导图引用：

```doc-block
id: roadmap
kind: mindmap
mode: reference
src: maps/roadmap.mindmap
---
```
