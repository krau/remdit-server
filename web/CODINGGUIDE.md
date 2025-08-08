# Coding Guide

这是一个协作文档编辑应用, 你需要完成编辑页面.

技术方案:

每个文档具有一个唯一ID, 通过调用后端 /api/file/:fileid 获取文档文本内容, 响应格式如下:

```json
{
    "fileid":"uuid",
    "content":"文档内容"
}
```

你需要在编辑页面中实现以下功能:

1. **加载文档内容**: 在编辑页面加载时, 通过文件ID从后端获取文档内容并显示在编辑器中.
2. **基于yjs+monaco的实时协作编辑**: 在编辑器中实现实时协作编辑功能, 后端具有 websocket 接口 `/api/ws/:room` , room id 即为文档的 ID
3. **保存文档内容**: 在编辑器中实现保存功能, 当用户点击保存按钮时, 将编辑器中的内容通过 POST 请求发送到后端的 `/api/file/:fileid` 接口, 请求体格式如下:

```json
{
    "content": "编辑后的文档内容"
}
```

前端技术栈:

Vue3+Vue Router+Shadcn UI+TailwindCSS4+Monaco Editor

注意事项:

- 确保界面友好, 提供必要的加载状态和错误提示.
- 使用 Shadcn UI 和 TailwindCSS4 进行样式设计, 确保界面美观, 响应式良好, 并具有暗色模式支持.
- 请求 API 时应直接使用相对路径, 例如 `/api/file/:fileid` 和 `/api/ws/:room` 等, 不要使用绝对路径.
- 执行命令时, 路径采用 git bash for windows 的格式, 例如 "/d/workdir/codes/remdit/server/web/src"