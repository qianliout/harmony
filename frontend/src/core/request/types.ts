// 与后端统一响应契约对齐的类型定义。
//
// 所有 REST 端点返回该结构，传输层恒为 HTTP 200，业务状态在 code 字段：
// - 成功：code=200，data 为单条（item）或列表，列表附带 totalItems/itemsPerPage/startIndex；
// - 业务失败：code=400/404/409/500，msg 为可展示文案、err 为错误详情。

/** 后端统一响应体。泛型 T 为 data 的元素类型。 */
export interface ApiResponse<T> {
  code: number
  msg: string
  err: string
  data?: T | T[] | null
  totalItems?: number
  itemsPerPage?: number
  startIndex?: number
}

/** 分页列表的展开结构，供表格分页组件直接绑定。 */
export interface Page<T> {
  items: T[]
  total: number
  itemsPerPage: number
  startIndex: number
}

/**
 * 分页查询参数基类，字段命名与后端 GET 参数对齐（page/pageSize）。
 * 各模块扩展自己的查询参数时继承它。
 */
export interface PageQuery {
  page: number
  pageSize: number
}

/** 通用枚举选项：value 为提交值（英文 key），label 为展示文案（中文）。 */
export interface EnumOption {
  value: string
  label: string
}
