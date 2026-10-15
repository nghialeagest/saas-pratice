import React, { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { ColumnDef } from "@tanstack/react-table";
import { listTenantsApi, updateTenantApi } from "@/api/tenants";
import { Tenant, TenantStatus } from "@/api/types";
import { useTenant } from "@/hooks/use-tenant";
import { Can } from "@/components/shared/Can";
import { StatusBadge } from "@/components/shared/StatusBadge";
import { DataTable } from "@/components/shared/DataTable";
import { CreateTenantDialog } from "./CreateTenantDialog";
import { useToast } from "@/components/ui/toast";
import { getLocalizedErrorMessage } from "@/lib/error-map";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Plus, Building2, ShieldCheck, PowerOff, ArrowRight } from "lucide-react";

export const TenantsPage: React.FC = () => {
  const { tenantId, setTenantId, refreshTenants } = useTenant();
  const { toast } = useToast();
  const queryClient = useQueryClient();

  const [page, setPage] = useState<number>(1);
  const pageSize = 10;
  const [statusFilter, setStatusFilter] = useState<string>("ALL");
  const [isCreateOpen, setIsCreateOpen] = useState<boolean>(false);

  // Query tenants list
  const {
    data: tenantsData,
    isLoading: isLoadingTenants,
    error: tenantsError,
  } = useQuery({
    queryKey: ["tenants", page, pageSize, statusFilter],
    queryFn: () =>
      listTenantsApi({
        page,
        pageSize,
        status: statusFilter === "ALL" ? undefined : statusFilter,
      }),
  });

  // Update tenant status mutation
  const updateStatusMutation = useMutation({
    mutationFn: ({ id, status }: { id: string; status: TenantStatus }) =>
      updateTenantApi(id, { id, status }),
    onSuccess: async (updated) => {
      queryClient.invalidateQueries({ queryKey: ["tenants"] });
      await refreshTenants();
      toast({
        title: "Cập nhật thành công",
        description: `Khách thuê ${updated.name} đã chuyển sang trạng thái ${updated.status}.`,
      });
    },
    onError: (err) => {
      toast({
        title: "Cập nhật thất bại",
        description: getLocalizedErrorMessage(err),
        variant: "destructive",
      });
    },
  });

  const columns: ColumnDef<Tenant, unknown>[] = [
    {
      accessorKey: "name",
      header: "Tên tổ chức",
      cell: ({ row }) => {
        const t = row.original;
        const isCurrent = t.id === tenantId;
        return (
          <div className="flex items-center gap-2">
            <div>
              <div className="font-medium text-foreground flex items-center gap-2">
                <span>{t.name}</span>
                {isCurrent && (
                  <span className="text-[10px] px-1.5 py-0.5 rounded bg-primary/10 text-primary font-semibold border border-primary/20">
                    Đang chọn
                  </span>
                )}
              </div>
              <div className="text-xs text-muted-foreground font-mono">
                Slug: {t.slug}
              </div>
            </div>
          </div>
        );
      },
    },
    {
      accessorKey: "status",
      header: "Trạng thái",
      cell: ({ row }) => <StatusBadge status={row.original.status} />,
    },
    {
      accessorKey: "createdAt",
      header: "Ngày khởi tạo",
      cell: ({ row }) => (
        <span className="text-xs text-muted-foreground">
          {row.original.createdAt.toLocaleDateString("vi-VN")}
        </span>
      ),
    },
    {
      id: "actions",
      header: "Thao tác",
      cell: ({ row }) => {
        const t = row.original;
        const isCurrent = t.id === tenantId;
        const isSuspended = t.status.toUpperCase() === "SUSPENDED";

        return (
          <div className="flex items-center gap-2">
            {!isCurrent && (
              <Button
                variant="outline"
                size="sm"
                className="h-8 gap-1 text-xs"
                onClick={() => {
                  setTenantId(t.id);
                  toast({
                    title: "Đã chuyển Không gian làm việc",
                    description: `Đang làm việc dưới phạm vi của "${t.name}".`,
                  });
                }}
              >
                <span>Chọn</span>
                <ArrowRight className="h-3.5 w-3.5" />
              </Button>
            )}

            <Can resource="tenant.tenant" action="write">
              <Button
                variant={isSuspended ? "outline" : "ghost"}
                size="sm"
                className={`h-8 gap-1 text-xs ${
                  isSuspended
                    ? "text-emerald-600 hover:text-emerald-700 hover:bg-emerald-50"
                    : "text-amber-600 hover:text-amber-700 hover:bg-amber-50"
                }`}
                disabled={updateStatusMutation.isPending}
                onClick={() => {
                  updateStatusMutation.mutate({
                    id: t.id,
                    status: isSuspended ? "ACTIVE" : "SUSPENDED",
                  });
                }}
              >
                {isSuspended ? (
                  <>
                    <ShieldCheck className="h-3.5 w-3.5" />
                    <span>Kích hoạt</span>
                  </>
                ) : (
                  <>
                    <PowerOff className="h-3.5 w-3.5" />
                    <span>Tạm đình chỉ</span>
                  </>
                )}
              </Button>
            </Can>
          </div>
        );
      },
    },
  ];

  const totalCount = tenantsData?.totalCount ?? 0;
  const totalPages = Math.ceil(totalCount / pageSize);

  return (
    <div className="space-y-6">
      {/* Top Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Quản lý Khách thuê (Tenants)</h1>
          <p className="text-sm text-muted-foreground mt-0.5">
            Quản trị phạm vi tổ chức và người thuê trên toàn hệ thống (Host Scope).
          </p>
        </div>

        <Can resource="tenant.tenant" action="write">
          <Button onClick={() => setIsCreateOpen(true)} className="gap-2">
            <Plus className="h-4 w-4" />
            Tạo khách thuê
          </Button>
        </Can>
      </div>

      {/* Metrics & Filter Bar */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2 space-y-0">
            <CardTitle className="text-sm font-medium">Tổng số khách thuê</CardTitle>
            <Building2 className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{totalCount}</div>
            <p className="text-xs text-muted-foreground mt-1">Toàn bộ khách thuê hệ thống</p>
          </CardContent>
        </Card>

        <div className="md:col-span-3 flex items-center justify-end gap-3">
          <span className="text-xs text-muted-foreground font-medium">Lọc theo trạng thái:</span>
          <div className="w-48">
            <Select
              value={statusFilter}
              onValueChange={(val) => {
                setStatusFilter(val);
                setPage(1);
              }}
            >
              <SelectTrigger className="h-9">
                <SelectValue placeholder="Trạng thái" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="ALL">Tất cả trạng thái</SelectItem>
                <SelectItem value="ACTIVE">ACTIVE (Hoạt động)</SelectItem>
                <SelectItem value="SUSPENDED">SUSPENDED (Tạm đình chỉ)</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
      </div>

      {/* Table */}
      {tenantsError ? (
        <div className="p-4 rounded-lg bg-destructive/10 text-destructive text-sm border border-destructive/20">
          {getLocalizedErrorMessage(tenantsError)}
        </div>
      ) : (
        <DataTable
          columns={columns}
          data={tenantsData?.tenants || []}
          isLoading={isLoadingTenants}
          emptyMessage="Chưa có khách thuê nào trong danh mục này."
          page={page}
          pageSize={pageSize}
          totalPages={totalPages}
          onPageChange={setPage}
        />
      )}

      {/* Create Dialog */}
      <CreateTenantDialog
        open={isCreateOpen}
        onOpenChange={setIsCreateOpen}
      />
    </div>
  );
};
