# 识别真实样张

本目录只在本机保存真实礼簿照片和人工标注，不入库。

每张照片配一个同名 `.expect.json`：

```json
{
  "page": "1-10",
  "pageTotal": 1070,
  "items": [
    { "name": "李成文", "amount": 50, "note": "11年国庆 孙女 100元 火炮20元" }
  ]
}
```

至少准备 5 张：横拍、竖拍、大量回礼标注、字迹潦草、本页合计与明细对不上各一张。
运行：

```bash
cd backend && go test -tags live -run TestRecognizeLive -v ./test/
```
