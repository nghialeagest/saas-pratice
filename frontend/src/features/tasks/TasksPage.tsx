import React, { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ColumnDef } from "@tanstack/react-table";
import { countTasksApi, getTaskApi } from "@/api/tasks";
import { Task } from "@/api/types";
import { useTenant } from "@/hooks/use-tenant";
import { Can } from "@/components/shared/Can";
import { StatusBadge } from "@/components/shared/StatusBadge";
import { DataTable } from "@/components/shared/DataTable";
import { CreateTaskDialog } from "./CreateTaskDialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useToast } from "@/components/ui/toast";
import { getLocalizedErrorMessage } from "@/lib/error-map";
import { Plus, CheckSquare, Camera, Search, AlertTriangle } from "lucide-react";

export const TasksPage: React.FC = () => {
  const { tenantId } = useTenant();
  const { toast } = useToast();

  const [isCreateOpen, setIsCreateOpen] = useState<boolean>(false);
  const [searchTaskId, setSearchTaskId] = useState<string>("");
  const [tasks, setTasks] = useState<Task[]>([]);
  const [isSearching, setIsSearching] = useState<boolean>(false);

  // Query tasks count
  const { data: countData, isLoading: isLoadingCount } = useQuery({
    queryKey: ["tasks-count", tenantId],
    queryFn: () => countTasksApi(),
    enabled: !!tenantId,
  });

  const handleSearchTask = async (e: React.FormEvent) => {
    e.preventDefault();
    const id = searchTaskId.trim();
    if (!id) return;

    try {
      setIsSearching(true);
      const task = await getTaskApi(id);
      // Add or update task in local list
      setTasks((prev) => {
        const filtered = prev.filter((t) => t.id !== task.id);
        return [task, ...filtered];
      });
      toast({
        title: "Đã tìm thấy công việc",
        description: `Mã: ${task.id} - Tiêu đề: ${task.title}`,
      });
      setSearchTaskId("");
    } catch (err) {
      toast({
        title: "Không tìm thấy công việc",
        description: getLocalizedErrorMessage(err),
        variant: "destructive",
      });
    } finally {
      setIsSearching(false);
    }
  };

  const columns: ColumnDef<Task, unknown>[] = [
    {
      accessorKey: "title",
      header: "Tiêu đề công việc",
      cell: ({ row }) => (
        <div>
          <div className="font-medium text-foreground">{row.original.title}</div>
          <div className="text-[11px] font-mono text-muted-foreground mt-0.5">
            ID: {row.original.id}
          </div>
        </div>
      ),
    },
    {
      accessorKey: "noteTitle",
      header: "Ghi chú liên kết (Snapshot)",
      cell: ({ row }) => {
        const noteTitle = row.original.noteTitle;
        if (!noteTitle) {
          return <span className="text-xs text-muted-foreground italic">Không có</span>;
        }

        return (
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <div className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded border border-blue-500/20 bg-blue-500/10 text-xs text-blue-700 dark:text-blue-300 cursor-help">
                  <Camera className="h-3 w-3 shrink-0" />
                  <span className="font-medium truncate max-w-[200px]">{noteTitle}</span>
                </div>
              </TooltipTrigger>
              <TooltipContent className="max-w-xs">
                <p className="font-semibold text-xs">Ảnh chụp (Snapshot)</p>
                <p className="text-[11px] text-muted-foreground">
                  Tiêu đề ghi chú tại thời điểm tạo công việc. Không bị ảnh hưởng nếu ghi chú gốc thay đổi.
                </p>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
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
      header: "Thời gian tạo",
      cell: ({ row }) => (
        <span className="text-xs text-muted-foreground">
          {row.original.createdAt.toLocaleString("vi-VN")}
        </span>
      ),
    },
  ];

  if (!tenantId) {
    return (
      <div className="flex flex-col items-center justify-center p-12 text-center border rounded-xl bg-card">
        <AlertTriangle className="h-10 w-10 text-amber-500 mb-3" />
        <h2 className="text-lg font-bold">Chưa chọn Không gian làm việc (Tenant)</h2>
        <p className="text-sm text-muted-foreground max-w-md mt-1">
          Các thao tác nghiệp vụ công việc yêu cầu phạm vi Tenant xác định. Vui lòng chọn một
          Tenant trên thanh tiêu đề.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Top Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Quản lý Công việc</h1>
          <p className="text-sm text-muted-foreground mt-0.5">
            Theo dõi nhiệm vụ và liên kết snapshot nội dung ghi chú.
          </p>
        </div>

        <Can resource="task.task" action="write">
          <Button onClick={() => setIsCreateOpen(true)} className="gap-2">
            <Plus className="h-4 w-4" />
            Tạo công việc
          </Button>
        </Can>
      </div>

      {/* Metrics & Search Bar */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2 space-y-0">
            <CardTitle className="text-sm font-medium">Tổng số công việc</CardTitle>
            <CheckSquare className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {isLoadingCount ? "..." : countData?.count ?? 0}
            </div>
            <p className="text-xs text-muted-foreground mt-1">
              Phạm vi tenant hiện tại
            </p>
          </CardContent>
        </Card>

        <Card className="md:col-span-2">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Tra cứu công việc theo ID</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSearchTask} className="flex gap-2">
              <Input
                placeholder="Nhập Task ID để tra cứu..."
                value={searchTaskId}
                onChange={(e) => setSearchTaskId(e.target.value)}
                disabled={isSearching}
                className="h-9 text-xs"
              />
              <Button
                type="submit"
                size="sm"
                variant="secondary"
                disabled={isSearching || !searchTaskId.trim()}
                className="gap-1.5 h-9 shrink-0"
              >
                <Search className="h-3.5 w-3.5" />
                <span>Tra cứu</span>
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>

      {/* Table */}
      <DataTable
        columns={columns}
        data={tasks}
        emptyMessage="Chưa có công việc nào được nạp trong phiên này. Hãy tạo công việc mới hoặc tra cứu theo ID."
      />

      {/* Create Dialog */}
      <CreateTaskDialog
        open={isCreateOpen}
        onOpenChange={setIsCreateOpen}
      />
    </div>
  );
};
