import React, { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { listUsersApi, deleteUserApi } from "@/api/users";
import { UserModel } from "@/api/types";
import { useTenant } from "@/hooks/use-tenant";
import { useAuth } from "@/hooks/use-auth";
import { Can } from "@/components/shared/Can";
import { useToast } from "@/components/ui/toast";
import { getLocalizedErrorMessage } from "@/lib/error-map";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { CreateUserDialog } from "./CreateUserDialog";
import { AssignRolesDialog } from "./AssignRolesDialog";
import { Plus, Users, ShieldCheck, Trash2, Search } from "lucide-react";

const PAGE_SIZE = 10;

export const UsersPage: React.FC = () => {
  const { tenantId } = useTenant();
  const { user: currentUser } = useAuth();
  const { toast } = useToast();
  const queryClient = useQueryClient();

  const [page, setPage] = useState(1);
  const [searchInput, setSearchInput] = useState("");
  const [search, setSearch] = useState("");
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [assigningUser, setAssigningUser] = useState<UserModel | null>(null);
  const [deletingUser, setDeletingUser] = useState<UserModel | null>(null);

  useEffect(() => {
    const t = setTimeout(() => {
      setSearch(searchInput);
      setPage(1);
    }, 300);
    return () => clearTimeout(t);
  }, [searchInput]);

  const { data, isLoading } = useQuery({
    queryKey: ["users", tenantId, page, search],
    queryFn: () => listUsersApi({ page, pageSize: PAGE_SIZE, search }),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteUserApi(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users"] });
      toast({ title: "Đã xoá người dùng" });
      setDeletingUser(null);
    },
    onError: (err) => {
      toast({
        title: "Xoá người dùng thất bại",
        description: getLocalizedErrorMessage(err),
        variant: "destructive",
      });
    },
  });

  const users = data?.items ?? [];

  if (!tenantId) {
    return (
      <Card>
        <CardContent className="py-12 text-center text-muted-foreground">
          Vui lòng chọn một tenant ở thanh tiêu đề để quản lý người dùng.
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold">Quản lý Người dùng</h1>
          <p className="text-sm text-muted-foreground">
            Thành viên của tenant hiện tại và vai trò của họ.
          </p>
        </div>
        <Can resource="user.user" action="write">
          <Button onClick={() => setIsCreateOpen(true)}>
            <Plus className="h-4 w-4 mr-2" />
            Thêm người dùng
          </Button>
        </Can>
      </div>

      <div className="flex items-end gap-4">
        <Card className="w-64">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Tổng số thành viên</CardTitle>
            <Users className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{users.length}</div>
            <p className="text-xs text-muted-foreground">Trang hiện tại</p>
          </CardContent>
        </Card>

        <div className="relative ml-auto w-72">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="pl-9"
            placeholder="Tìm theo tên đăng nhập hoặc email..."
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
          />
        </div>
      </div>

      <div className="rounded-md border">
        <table className="w-full text-sm">
          <thead className="border-b bg-muted/40">
            <tr className="text-left">
              <th className="px-4 py-3 font-medium">Tên đăng nhập</th>
              <th className="px-4 py-3 font-medium">Email</th>
              <th className="px-4 py-3 font-medium">Vai trò</th>
              <th className="px-4 py-3 font-medium">Ngày tạo</th>
              <th className="px-4 py-3 font-medium text-right">Thao tác</th>
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              [...Array(3)].map((_, i) => (
                <tr key={i} className="border-b">
                  <td colSpan={5} className="px-4 py-3">
                    <Skeleton className="h-6 w-full" />
                  </td>
                </tr>
              ))
            ) : users.length === 0 ? (
              <tr>
                <td colSpan={5} className="px-4 py-10 text-center text-muted-foreground">
                  Chưa có người dùng nào trong tenant này.
                </td>
              </tr>
            ) : (
              users.map((u) => {
                const isSelf = u.id === currentUser?.id;
                return (
                  <tr key={u.id} className="border-b last:border-0 hover:bg-muted/30">
                    <td className="px-4 py-3">
                      <div className="font-medium">{u.username}</div>
                      {u.name && <div className="text-xs text-muted-foreground">{u.name}</div>}
                    </td>
                    <td className="px-4 py-3">
                      <div>{u.email}</div>
                      {!u.emailConfirmed && (
                        <Badge variant="outline" className="mt-1 text-xs">
                          chưa xác thực
                        </Badge>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex flex-wrap gap-1">
                        {u.roles.length === 0 ? (
                          <span className="text-xs text-muted-foreground">—</span>
                        ) : (
                          u.roles.map((r) => (
                            <Badge key={r.id} variant="secondary">
                              {r.name}
                            </Badge>
                          ))
                        )}
                      </div>
                    </td>
                    <td className="px-4 py-3 text-muted-foreground">
                      {u.createdAt.toLocaleDateString("vi-VN")}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center justify-end gap-2">
                        <Can resource="user.user" action="write">
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => setAssigningUser(u)}
                          >
                            <ShieldCheck className="h-3.5 w-3.5 mr-1.5" />
                            Vai trò
                          </Button>
                        </Can>
                        <Can resource="user.user" action="write">
                          <Button
                            variant="ghost"
                            size="sm"
                            className="text-destructive"
                            disabled={isSelf}
                            title={isSelf ? "Không thể tự xoá tài khoản của mình" : undefined}
                            onClick={() => setDeletingUser(u)}
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </Button>
                        </Can>
                      </div>
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>

      <div className="flex items-center justify-between">
        <span className="text-sm text-muted-foreground">Trang {page}</span>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage(page - 1)}>
            Trước
          </Button>
          <Button
            variant="outline"
            size="sm"
            disabled={users.length < PAGE_SIZE}
            onClick={() => setPage(page + 1)}
          >
            Sau
          </Button>
        </div>
      </div>

      <CreateUserDialog open={isCreateOpen} onOpenChange={setIsCreateOpen} />
      <AssignRolesDialog user={assigningUser} onOpenChange={() => setAssigningUser(null)} />

      <AlertDialog open={!!deletingUser} onOpenChange={() => setDeletingUser(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Xoá người dùng?</AlertDialogTitle>
            <AlertDialogDescription>
              Tài khoản <span className="font-medium">{deletingUser?.username}</span> sẽ bị xoá.
              Thao tác này không thể hoàn tác.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Huỷ</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deletingUser && deleteMutation.mutate(deletingUser.id)}
            >
              Xoá
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
};
