/** 仅用于不影响功能边界的展示差异；识别等能力边界必须用条件编译。 */
export const isH5 = (() => {
  // #ifdef H5
  return true
  // #endif
  // #ifndef H5
  return false
  // #endif
})()
