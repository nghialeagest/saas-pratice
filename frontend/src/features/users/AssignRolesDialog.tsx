import React, { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { listRolesApi } from "@/api/roles";
import { assignRolesApi } from "@/api/users";
import { UserModel } from "@/api/types";
import { useTenant } from "@/hooks/use-tenant";
import { useAuth } from "@/hooks/use-auth";
import { useToast } from "@/components/ui/toast";
import { getLocalizedErrorMessage } from "@/lib/error-map";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { AlertTriangle } from "lucide-react";

interface Props {
  user: UserModel | null;
  onOpenChange: (open: boolean) => void;
}

export const AssignRolesDialog: React.FC<Props> = ({ user, onOpenChange }) => {
  const { tenantId } = useTenant();
  const { user: currentUser } = useAuth();
  const { toast } = useToast();
  const queryClient = useQueryClient();

  const [selected, setSelected] = useState<string[]>([]);

  const { data: roles, isLoading } = useQuery({
    queryKey: ["roles", tenantId],
    queryFn: listRolesApi,
    enabled: !!user,
  });

  useEffect(() => {
    setSelected(user ? user.roles.map((r) => r.id) : []);
  }, [user]);

  const isSelf = !!user && !!currentUser && user.id === currentUser.id;
  const losingRoles = isSelf && selected.length < (user?.roles.length ?? 0);

  const mutation = useMutation({
    mutationFn: (roleIds: string[]) => assignRolesApi(user!.id, roleIds),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users"] });
      toast({
        title: "Cập nhật vai trò thành công",
        description: `Vai trò của "${user?.username}" đã được lưu.`,
      });
      onOpenChange(false);
    },
    onError: (err) => {
      toast({
        title: "Cập nhật vai trò thất bại",
        description: getLocalizedErrorMessage(err),
        variant: "destructive",
      });
    },
  });

  const toggle = (id: string) => {
    setSelected((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]));
  };

  return (
    <Dialog open={!!user} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Gán vai trò</DialogTitle>
          <DialogDescription>
            Vai trò áp dụng trong phạm vi tenant đang chọn. Người dùng{" "}
            <span className="font-medium">{user?.username}</span>.
          </DialogDescription>
        </DialogHeader>

        {isLoading ? (
          <div className="space-y-2">
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-full" />
          </div>
        ) : (roles?.length ?? 0) === 0 ? (
          <p className="text-sm text-muted-foreground py-4">
            Tenant này chưa có vai trò nào. Hãy tạo vai trò ở trang "Vai trò &amp; Quyền".
          </p>
        ) : (
          <div className="space-y-2 max-h-72 overflow-y-auto py-2">
            {roles!.map((role) => (
              <label
                key={role.id}
                className="flex items-center gap-3 rounded-md border p-3 cursor-pointer hover:bg-accent"
              >
                <Checkbox
                  checked={selected.includes(role.id)}
                  onCheckedChange={() => toggle(role.id)}
                />
                <span className="text-sm font-medium">{role.name}</span>
              </label>
            ))}
          </div>
        )}

        {losingRoles && (
          <div className="flex gap-2 rounded-md border border-amber-300 bg-amber-50 p-3 text-sm text-amber-900">
            <AlertTriangle className="h-4 w-4 shrink-0 mt-0.5" />
            <span>
              Bạn đang gỡ vai trò của chính mình. Thao tác này có thể khiến bạn mất quyền truy
              cập vào tenant hiện tại.
            </span>
          </div>
        )}

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Huỷ
          </Button>
          <Button onClick={() => mutation.mutate(selected)} disabled={mutation.isPending}>
            {mutation.isPending ? "Đang lưu..." : "Lưu vai trò"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
