“原本的计划”是指直接提供接口文档，但由于您补充了**鉴权**和**分页**的新需求，原有的后端代码不再满足要求，因此我需要先修改代码，再提供文档。

更新后的完整执行计划如下：

1. **修改后端代码**

   * **Repository**: 修改 `internal/repository/comment.go`，增加 `GetByArticleIDWithPagination` 方法，实现“根评论分页 + 加载所有子评论”的逻辑。

   * **Handler**: 修改 `internal/handler/comment.go`，让 `GetComments` 支持 `page`/`page_size` 参数，并调用分页方法。

   * **Router**: 修改 `internal/router/router.go`，为所有评论接口（增删查）启用 `HandleWithAuth`，强制验证 Token。

2. **更新文档**

   * 修改代码中的 Swagger 注释（添加 Auth 和分页参数说明）。

   * 执行 `swag init` 更新文档。

3. **交付接口**

   * 直接输出最终的接口文档（URL、Header、Params、Response），供您对接前端。

