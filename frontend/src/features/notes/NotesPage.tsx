import React, { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { ColumnDef } from "@tanstack/react-table";
import { listNotesApi, countNotesApi, publishNoteApi } from "@/api/notes";
import { Note } from "@/api/types";
import { useTenant } from "@/hooks/use-tenant";
import { Can } from "@/components/shared/Can";
import { StatusBadge } from "@/components/shared/StatusBadge";
import { DataTable } from "@/components/shared/DataTable";
import { CreateNoteDialog } from "./CreateNoteDialog";
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
import { Plus, Send, FileText, AlertTriangle } from "lucide-react";

export const NotesPage: React.FC = () => {
  const { tenantId } = useTenant();
  const { toast } = useToast();
  const queryClient = useQueryClient();

  const [page, setPage] = useState<number>(1);
  const pageSize = 10;
  const [statusFilter, setStatusFilter] = useState<string>("ALL");
  const [isCreateOpen, setIsCreateOpen] = useState<boolean>(false);
  const [publishingNote, setPublishingNote] = useState<Note | null>(null);

  // Query notes list
  const {
    data: notesData,
    isLoading: isLoadingNotes,
    error: notesError,
  } = useQuery({
    queryKey: ["notes", tenantId, page, pageSize, statusFilter],
    queryFn: () =>
      listNotesApi({
        page,
        pageSize,
        status: statusFilter === "ALL" ? undefined : statusFilter,
      }),
    enabled: !!tenantId,
  });

  // Query notes count
  const { data: countData } = useQuery({
    queryKey: ["notes-count", tenantId, statusFilter],
    queryFn: () =>
      countNotesApi(statusFilter === "ALL" ? undefined : statusFilter),
    enabled: !!tenantId,
  });

  // Publish mutation
  const publishMutation = useMutation({
    mutationFn: (id: string) => publishNoteApi(id),
    onSuccess: (updated) => {
      queryClient.invalidateQueries({ queryKey: ["notes"] });
      queryClient.invalidateQueries({ queryKey: ["notes-count"] });
      toast({
        title: "Xuất bản thành công",
        description: `Ghi chú "${updated.title}" đã được chuyển sang trạng thái PUBLISHED.`,
      });
      setPublishingNote(null);
    },
    onError: (err) => {
      toast({
        title: "Xuất bản thất bại",
        description: getLocalizedErrorMessage(err),
        variant: "destructive",
      });
      setPublishingNote(null);
    },
  });

  const columns: ColumnDef<Note, unknown>[] = [
    {
      accessorKey: "title",
      header: "Tiêu đề",
      cell: ({ row }) => {
        const note = row.original;
        return (
          <div>
            <div className="font-medium text-foreground">{note.title}</div>
            {note.body && (
              <div className="text-xs text-muted-foreground line-clamp-1 mt-0.5">
                {note.body}
              </div>
            )}
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
      header: "Thời gian tạo",
      cell: ({ row }) => {
        const d = row.original.createdAt;
        return (
          <span className="text-xs text-muted-foreground">
            {d ? d.toLocaleString("vi-VN") : "—"}
          </span>
        );
      },
    },
    {
      id: "actions",
      header: "Hành động",
      cell: ({ row }) => {
        const note = row.original;
        if (note.status !== "DRAFT") return null;

        return (
          <Can resource="note.note" action="write">
            <Button
              variant="outline"
              size="sm"
              className="h-8 gap-1 text-xs"
              onClick={() => setPublishingNote(note)}
            >
              <Send className="h-3 w-3" />
              Xuất bản
            </Button>
          </Can>
        );
      },
    },
  ];

  if (!tenantId) {
    return (
      <div className="flex flex-col items-center justify-center p-12 text-center border rounded-xl bg-card">
        <AlertTriangle className="h-10 w-10 text-amber-500 mb-3" />
        <h2 className="text-lg font-bold">Chưa chọn Không gian làm việc (Tenant)</h2>
        <p className="text-sm text-muted-foreground max-w-md mt-1">
          Các thao tác nghiệp vụ ghi chú yêu cầu phạm vi Tenant xác định. Vui lòng chọn một
          Tenant trên thanh tiêu đề hoặc tạo Tenant mới tại trang Khách thuê.
        </p>
      </div>
    );
  }

  const totalCount = notesData?.totalCount ?? countData?.count ?? 0;
  const totalPages = Math.ceil(totalCount / pageSize);

  return (
    <div className="space-y-6">
      {/* Top Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Quản lý Ghi chú</h1>
          <p className="text-sm text-muted-foreground mt-0.5">
            Xem và phát hành các tài liệu, ghi chú của tenant.
          </p>
        </div>

        <Can resource="note.note" action="write">
          <Button onClick={() => setIsCreateOpen(true)} className="gap-2">
            <Plus className="h-4 w-4" />
            Tạo ghi chú
          </Button>
        </Can>
      </div>

      {/* Metrics & Filter Bar */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2 space-y-0">
            <CardTitle className="text-sm font-medium">Tổng số ghi chú</CardTitle>
            <FileText className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{totalCount}</div>
            <p className="text-xs text-muted-foreground mt-1">
              Phạm vi tenant hiện tại
            </p>
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
                <SelectItem value="DRAFT">DRAFT (Bản nháp)</SelectItem>
                <SelectItem value="PUBLISHED">PUBLISHED (Đã phát hành)</SelectItem>
                <SelectItem value="ARCHIVED">ARCHIVED (Lưu trữ)</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
      </div>

      {/* Table */}
      {notesError ? (
        <div className="p-4 rounded-lg bg-destructive/10 text-destructive text-sm border border-destructive/20">
          {getLocalizedErrorMessage(notesError)}
        </div>
      ) : (
        <DataTable
          columns={columns}
          data={notesData?.notes || []}
          isLoading={isLoadingNotes}
          emptyMessage="Chưa có ghi chú nào trong danh mục này."
          page={page}
          pageSize={pageSize}
          totalPages={totalPages}
          onPageChange={setPage}
        />
      )}

      {/* Create Dialog */}
      <CreateNoteDialog
        open={isCreateOpen}
        onOpenChange={setIsCreateOpen}
      />

      {/* Confirm Publish Alert Dialog */}
      <AlertDialog
        open={!!publishingNote}
        onOpenChange={(open) => !open && setPublishingNote(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Xác nhận xuất bản ghi chú</AlertDialogTitle>
            <AlertDialogDescription>
              Bạn có chắc chắn muốn xuất bản ghi chú &quot;{publishingNote?.title}&quot;? Sau
              khi xuất bản, tài liệu sẽ chuyển sang trạng thái PUBLISHED.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={publishMutation.isPending}>
              Huỷ
            </AlertDialogCancel>
            <AlertDialogAction
              disabled={publishMutation.isPending}
              onClick={() => {
                if (publishingNote) {
                  publishMutation.mutate(publishingNote.id);
                }
              }}
            >
              {publishMutation.isPending ? "Đang xuất bản..." : "Xác nhận xuất bản"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
};
