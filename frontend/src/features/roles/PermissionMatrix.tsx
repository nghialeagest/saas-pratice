import React, { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { getRolePermissionsApi, setRolePermissionsApi } from "@/api/roles";
import { listPermissionCatalogueApi } from "@/api/permissions";
import { Permission, Role, permissionKey } from "@/api/types";
import { useTenant } from "@/hooks/use-tenant";
import { Can } from "@/components/shared/Can";
import { useToast } from "@/components/ui/toast";
import { getLocalizedErrorMessage } from "@/lib/error-map";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { Save, RotateCcw, Lock } from "lucide-react";

// Display names only. The permission list itself always comes from the server.
const NAMESPACE_LABELS: Record<string, string> = {
  "note.note": "Ghi chú",
  "task.task": "Công việc",
  "user.user": "Người dùng",
  "user.role": "Vai trò",
  "tenant.tenant": "Khách thuê",
};

const ACTION_LABELS: Record<string, string> = {
  read: "Xem",
  write: "Chỉnh sửa",
  "*": "Toàn quyền",
};

interface Props {
  role: Role;
}

export const PermissionMatrix: React.FC<Props> = ({ role }) => {
  const { tenantId } = useTenant();
  const { toast } = useToast();
  const queryClient = useQueryClient();

  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [initial, setInitial] = useState<Set<string>>(new Set());

  const { data: catalogue, isLoading: loadingCatalogue } = useQuery({
    queryKey: ["permission-catalogue"],
    queryFn: listPermissionCatalogueApi,
  });

  const { data: granted, isLoading: loadingGranted } = useQuery({
    queryKey: ["role-permissions", tenantId, role.id],
    queryFn: () => getRolePermissionsApi(role.id),
  });

  useEffect(() => {
    if (!granted) return;
    const keys = new Set(granted.map(permissionKey));
    setSelected(new Set(keys));
    setInitial(new Set(keys));
  }, [granted, role.id]);

  const grouped = useMemo(() => {
    const map = new Map<string, Permission[]>();
    (catalogue ?? []).forEach((p) => {
      const list = map.get(p.namespace) ?? [];
      list.push(p);
      map.set(p.namespace, list);
    });
    return Array.from(map.entries());
  }, [catalogue]);

  const isDirty = useMemo(() => {
    if (selected.size !== initial.size) return true;
    for (const k of selected) if (!initial.has(k)) return true;
    return false;
  }, [selected, initial]);

  // Host-only permissions cannot take effect inside a tenant, so they are locked.
  const isLocked = (p: Permission) => !!p.host_only && !!tenantId;

  const toggle = (p: Permission) => {
    if (isLocked(p)) return;
    const key = permissionKey(p);
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  const toggleNamespace = (perms: Permission[]) => {
    const usable = perms.filter((p) => !isLocked(p));
    const allOn = usable.every((p) => selected.has(permissionKey(p)));
    setSelected((prev) => {
      const next = new Set(prev);
      usable.forEach((p) => {
        if (allOn) next.delete(permissionKey(p));
        else next.add(permissionKey(p));
      });
      return next;
    });
  };

  const mutation = useMutation({
    mutationFn: () => {
      const perms = (catalogue ?? []).filter((p) =>
        selected.has(permissionKey(p)),
      );
      return setRolePermissionsApi(role.id, perms);
    },
    onSuccess: () => {
      setInitial(new Set(selected));
      queryClient.invalidateQueries({
        queryKey: ["role-permissions", tenantId, role.id],
      });
      toast({
        title: "Lưu quyền thành công",
        description: `Vai trò "${role.name}" đã được cập nhật.`,
      });
    },
    onError: (err) => {
      toast({
        title: "Lưu quyền thất bại",
        description: getLocalizedErrorMessage(err),
        variant: "destructive",
      });
    },
  });

  if (loadingCatalogue || loadingGranted) {
    return (
      <div className="space-y-3">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-24 w-full" />
      </div>
    );
  }

  return (
    <TooltipProvider>
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="font-semibold">Quyền của vai trò "{role.name}"</h3>
            <p className="text-xs text-muted-foreground">
              {selected.size} quyền được chọn
              {isDirty && (
                <span className="text-amber-600"> · có thay đổi chưa lưu</span>
              )}
            </p>
          </div>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={!isDirty}
              onClick={() => setSelected(new Set(initial))}
            >
              <RotateCcw className="h-3.5 w-3.5 mr-1.5" />
              Hoàn tác
            </Button>
            <Can resource="user.role" action="write">
              <Button
                size="sm"
                disabled={!isDirty || mutation.isPending}
                onClick={() => mutation.mutate()}
              >
                <Save className="h-3.5 w-3.5 mr-1.5" />
                {mutation.isPending ? "Đang lưu..." : "Lưu"}
              </Button>
            </Can>
          </div>
        </div>

        <div className="space-y-3">
          {grouped.map(([namespace, perms]) => {
            const usable = perms.filter((p) => !isLocked(p));
            const onCount = usable.filter((p) =>
              selected.has(permissionKey(p)),
            ).length;
            const groupState =
              onCount === 0
                ? false
                : onCount === usable.length
                  ? true
                  : "indeterminate";

            return (
              <div key={namespace} className="rounded-md border p-4">
                <label className="flex items-center gap-3 pb-3 border-b cursor-pointer">
                  <Checkbox
                    checked={groupState}
                    onCheckedChange={() => toggleNamespace(perms)}
                    disabled={usable.length === 0}
                  />
                  <div>
                    <div className="text-sm font-medium">
                      {NAMESPACE_LABELS[namespace] ?? namespace}
                    </div>
                    <div className="text-xs text-muted-foreground font-mono">
                      {namespace}
                    </div>
                  </div>
                </label>

                <div className="grid grid-cols-2 sm:grid-cols-3 gap-2 pt-3">
                  {perms.map((p) => {
                    const key = permissionKey(p);
                    const locked = isLocked(p);
                    const row = (
                      <label
                        className={`flex items-center gap-2 rounded px-2 py-1.5 text-sm ${
                          locked
                            ? "opacity-50 cursor-not-allowed"
                            : "cursor-pointer hover:bg-accent"
                        }`}
                      >
                        <Checkbox
                          checked={selected.has(key)}
                          disabled={locked}
                          onCheckedChange={() => toggle(p)}
                        />
                        <span>{ACTION_LABELS[p.action] ?? p.action}</span>
                        {locked && (
                          <Lock className="h-3 w-3 text-muted-foreground" />
                        )}
                      </label>
                    );

                    return locked ? (
                      <Tooltip key={key}>
                        <TooltipTrigger asChild>
                          <div>{row}</div>
                        </TooltipTrigger>
                        <TooltipContent>
                          Quyền này chỉ dùng được ở phạm vi hệ thống (host
                          scope)
                        </TooltipContent>
                      </Tooltip>
                    ) : (
                      <div key={key}>{row}</div>
                    );
                  })}
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </TooltipProvider>
  );
};
