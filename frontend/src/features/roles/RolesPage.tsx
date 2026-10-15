import React, { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { listRolesApi, deleteRoleApi } from "@/api/roles";
import { Role } from "@/api/types";
import { useTenant } from "@/hooks/use-tenant";
import { Can } from "@/components/shared/Can";
import { useToast } from "@/components/ui/toast";
import { getLocalizedErrorMessage } from "@/lib/error-map";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
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
import { CreateRoleDialog } from "./CreateRoleDialog";
import { PermissionMatrix } from "./PermissionMatrix";
import { Plus, Shield, Trash2 } from "lucide-react";

export const RolesPage: React.FC = () => {
  const { tenantId } = useTenant();
  const { toast } = useToast();
  const queryClient = useQueryClient();

  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [deleting, setDeleting] = useState<Role | null>(null);

  const { data: roles, isLoading } = useQuery({
    queryKey: ["roles", tenantId],
    queryFn: listRolesApi,
  });

  useEffect(() => {
    if (!roles || roles.length === 0) {
      setSelectedId(null);
      return;
    }
    if (!selectedId || !roles.some((r) => r.id === selectedId)) {
      setSelectedId(roles[0].id);
    }
  }, [roles, selectedId]);

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteRoleApi(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["roles", tenantId] });
      toast({ title: "Đã xoá vai trò" });
      setDeleting(null);
      setSelectedId(null);
    },
    onError: (err) => {
      toast({
        title: "Xoá vai trò thất bại",
        description: getLocalizedErrorMessage(err),
        variant: "destructive",
      });
    },
  });

  const selectedRole = roles?.find((r) => r.id === selectedId) ?? null;

  if (!tenantId) {
    return (
      <Card>
        <CardContent className="py-12 text-center text-muted-foreground">
          Vui lòng chọn một tenant ở thanh tiêu đề. Vai trò và quyền thuộc về từng tenant.
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold">Vai trò &amp; Quyền</h1>
          <p className="text-sm text-muted-foreground">
            Danh mục quyền được sinh tự động từ các module đang chạy.
          </p>
        </div>
        <Can resource="user.role" action="write">
          <Button onClick={() => setIsCreateOpen(true)}>
            <Plus className="h-4 w-4 mr-2" />
            Tạo vai trò
          </Button>
        </Can>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-[280px_1fr] gap-6">
        <div className="space-y-2">
          {isLoading ? (
            [...Array(3)].map((_, i) => <Skeleton key={i} className="h-14 w-full" />)
          ) : (roles?.length ?? 0) === 0 ? (
            <Card>
              <CardContent className="py-8 text-center text-sm text-muted-foreground">
                Chưa có vai trò nào.
              </CardContent>
            </Card>
          ) : (
            roles!.map((role) => (
              <div
                key={role.id}
                onClick={() => setSelectedId(role.id)}
                className={`flex items-center justify-between rounded-md border p-3 cursor-pointer transition-colors ${
                  role.id === selectedId ? "border-primary bg-accent" : "hover:bg-muted/50"
                }`}
              >
                <div className="flex items-center gap-2">
                  <Shield className="h-4 w-4 text-muted-foreground" />
                  <span className="text-sm font-medium">{role.name}</span>
                </div>
                <Can resource="user.role" action="write">
                  <Button
                    variant="ghost"
                    size="sm"
                    className="text-destructive h-7 w-7 p-0"
                    disabled={role.preserved}
                    title={role.preserved ? "Vai trò hệ thống, không thể xoá" : undefined}
                    onClick={(e) => {
                      e.stopPropagation();
                      setDeleting(role);
                    }}
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                </Can>
              </div>
            ))
          )}
        </div>

        <Card>
          <CardContent className="pt-6">
            {selectedRole ? (
              <PermissionMatrix key={selectedRole.id} role={selectedRole} />
            ) : (
              <div className="py-12 text-center text-muted-foreground text-sm">
                Chọn một vai trò ở cột bên trái để xem và chỉnh quyền.
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      <CreateRoleDialog open={isCreateOpen} onOpenChange={setIsCreateOpen} />

      <AlertDialog open={!!deleting} onOpenChange={() => setDeleting(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Xoá vai trò?</AlertDialogTitle>
            <AlertDialogDescription>
              Vai trò <span className="font-medium">{deleting?.name}</span> sẽ bị xoá. Người dùng
              đang mang vai trò này sẽ mất các quyền tương ứng.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Huỷ</AlertDialogCancel>
            <AlertDialogAction onClick={() => deleting && deleteMutation.mutate(deleting.id)}>
              Xoá
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
};
