# saas-base

Bộ khung multi-tenant SaaS, rút ra từ việc phân tích [go-saas/kit](https://github.com/go-saas/kit)
— giữ lại các pattern tốt, loại bỏ phần đã lỗi thời, và vá những lỗi tìm được trong quá trình đọc.

```bash
go run .          # xem module nào đã nạp
go test ./...     # chạy bộ test cách ly tenant
```

## Cấu trúc

```
platform/                   ← hạ tầng dùng chung, không chứa nghiệp vụ
├── module/module.go           hợp đồng Module + sổ đăng ký
└── tenant/tenant.go           cách ly dữ liệu đa tenant (gorm callbacks)

modules/<tên>/              ← mỗi module nghiệp vụ một thư mục
├── api/                       CÔNG KHAI — module khác chỉ được import đây
│   └── api.go                 DTO + interface + hằng số dùng chung + Permissions()
├── internal/                  RIÊNG TƯ — Go compiler tự chặn import từ ngoài
│   ├── biz/                   entity gorm + interface repo + logic nghiệp vụ
│   ├── data/                  cài đặt repo bằng gorm
│   └── service/               cài đặt interface công khai + mapper DTO↔entity
└── module.go                  bản khai của module + hàm New()

main.go                     ← composition root
tenant_isolation_test.go    ← bộ test bắt buộc, chạy trước mọi thứ khác 
```

## Thêm module mới

```bash
cp -r modules/note modules/order
```

Sửa 4 chỗ:

1. Tên package trong `module.go`
2. `api/api.go` — DTO, interface công khai, hằng số, `Permissions()`
3. `internal/biz` — entity (nhúng `tenant.Model` nếu thuộc về tenant)
4. `module.go` — `Name()` và `Tables()`

Rồi thêm một dòng vào `main.go`:

```go
_ "saasbase/modules/order"
```

Tháo module = xoá đúng dòng đó. Không sửa gì thêm.

## Bốn luật bất di bất dịch

**1. Module chỉ import `api/` của module khác.**
`internal/` được Go compiler chặn tự động — thử vi phạm sẽ nhận:

```
use of internal package saasbase/modules/note/internal/biz not allowed
```

**2. Hằng số enum dùng chung phải nằm ở `api/`.**
Đây là nguyên nhân của cả 3 chỗ rò ranh giới trong go-saas/kit — hằng enum nằm nhầm ở
`private/biz` khiến module khác buộc phải import package riêng tư.

**3. `New()` trả về interface, không trả struct nội bộ.**
Nhờ vậy module khác không có cách nào chạm vào `internal/`.

**4. Entity thuộc tenant thì nhúng `tenant.Model`. Entity toàn cục thì không.**
Ví dụ: `Note` thuộc tenant; một `User` dùng chung nhiều tenant thì không nhúng.

## Cách ly tenant

`platform/tenant` đăng ký 3 gorm callback, chạy tự động cho mọi entity có `TenantID`:

| Callback | Làm gì |
|---|---|
| `tenant:assign` | gán `tenant_id` từ context khi INSERT — **vô điều kiện** |
| `tenant:filter` | thêm `WHERE tenant_id = ?` khi SELECT |
| `tenant:filter_delete` | thêm `WHERE tenant_id = ?` khi DELETE |

Tầng `data/` không có một dòng nào đụng tới `tenant_id` — callback lo hết.

Muốn truy vấn xuyên tenant (host admin) thì phải **chủ động** gọi:

```go
ctx = tenant.SkipFilter(ctx)
```

Hàm này cố ý không có đường nào đi từ HTTP header vào — khác với kit, nơi cờ tương đương
bật được qua header `internal-tenant-data-filter`.

## Bộ test cách ly

`tenant_isolation_test.go` có 5 assert. Giữ nó xanh trước khi viết bất cứ nghiệp vụ nào.

| Assert | Bắt lỗi gì |
|---|---|
| mỗi tenant chỉ thấy dữ liệu của mình | rò khi đọc |
| ghi vào A không đổi B | rò khi ghi |
| không bản ghi nào có `tenant_id` rỗng | bug gán thiếu tenant khi INSERT |
| host context không thấy dữ liệu tenant | rò ngược lên host |
| chỉ khi `SkipFilter` mới thấy toàn bộ | filter có thật sự hoạt động |

Assert thứ ba là assert mà go-saas/kit không có — và vì không có nên bug
`tenant_id = NULL` ở luồng checkout sống sót trong một repo 13.549 node.

## Còn thiếu

Template này mới có tầng dữ liệu và cơ chế module. Cần bổ sung theo thứ tự:

1. **Tầng HTTP** — Kratos (nếu đi proto/gRPC) hoặc Gin/chi (nếu monolith)
2. **Authz** — casbin với model 5 chiều `sub, namespace, obj, act, ten`;
   nhớ cache enforcer theo tenant, đừng dựng mới mỗi request
3. **Identity** — `User` toàn cục, `UserTenant` bảng nối có trạng thái,
   `Role` unique theo `(tenant_id, normalized_name)`, `UserRole` có `tenant_id` trong khoá chính
4. **`ConnStrResolver`** — nếu cần hỗ trợ tenant dùng DB riêng
5. **Driver DB thật** — hiện dùng sqlite in-memory cho test
