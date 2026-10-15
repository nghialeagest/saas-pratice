import { AxiosError } from "axios";
import { AppErrorResponse } from "@/api/types";

const ERROR_CODE_MAP: Record<string, string> = {
  "user.invalid_credentials": "Tên đăng nhập hoặc mật khẩu không chính xác.",
  "user.not_found": "Không tìm thấy người dùng.",
  "user.locked_out": "Tài khoản tạm thời bị khóa do nhập sai mật khẩu nhiều lần. Vui lòng thử lại sau.",
  "user.inactive": "Tài khoản hiện đang bị vô hiệu hóa.",
  "tenant.forbidden": "Bạn không có quyền truy cập vào tổ chức này.",
  "tenant.not_found": "Không tìm thấy thông tin tổ chức.",
  "tenant.slug_conflict": "Mã định danh (slug) tổ chức đã tồn tại, vui lòng chọn mã khác.",
  "tenant.invalid_argument": "Dữ liệu tổ chức không hợp lệ.",
  "user.already_exists": "Tên đăng nhập hoặc email đã được sử dụng.",
  "user.invalid_argument": "Dữ liệu người dùng không hợp lệ.",
  "role.not_found": "Không tìm thấy vai trò.",
  "role.invalid_argument": "Dữ liệu vai trò không hợp lệ.",
  "authz.permission_denied": "Bạn không có quyền thực hiện thao tác này.",
  "note.not_found": "Ghi chú không tồn tại hoặc bạn không có quyền truy cập.",
  "task.not_found": "Công việc không tồn tại hoặc bạn không có quyền truy cập.",
  "internal": "Đã xảy ra lỗi máy chủ, vui lòng thử lại sau.",
};

export const getErrorMessage = (error: unknown): string => {
  if (!error) return "Đã xảy ra lỗi không xác định.";

  if (error instanceof AxiosError) {
    const data = error.response?.data as AppErrorResponse | undefined;
    if (data?.code && ERROR_CODE_MAP[data.code]) {
      return ERROR_CODE_MAP[data.code];
    }

    if (error.response?.status === 403) {
      return "Truy cập bị từ chối: Bạn không có quyền truy cập vào tài nguyên này.";
    }
    if (error.response?.status === 401) {
      return "Phiên đăng nhập đã hết hạn hoặc không hợp lệ. Vui lòng đăng nhập lại.";
    }
    if (error.response?.status === 404) {
      return "Tài nguyên được yêu cầu không tìm thấy.";
    }
    if (error.response?.status === 400) {
      return "Yêu cầu không hợp lệ. Vui lòng kiểm tra lại thông tin.";
    }
  }

  return "Đã xảy ra lỗi kết nối, vui lòng thử lại.";
};

export const getLocalizedErrorMessage = getErrorMessage;
