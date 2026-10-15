import React from "react";
import { useForm, Controller } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createTaskApi } from "@/api/tasks";
import { listNotesApi } from "@/api/notes";
import { useTenant } from "@/hooks/use-tenant";
import { getLocalizedErrorMessage } from "@/lib/error-map";
import { useToast } from "@/components/ui/toast";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Camera, Info } from "lucide-react";

const taskSchema = z.object({
  title: z.string().min(1, "Tiêu đề công việc không được để trống"),
  note_id: z.string().optional(),
});

type TaskFormData = z.infer<typeof taskSchema>;

interface CreateTaskDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export const CreateTaskDialog: React.FC<CreateTaskDialogProps> = ({
  open,
  onOpenChange,
}) => {
  const { tenantId } = useTenant();
  const queryClient = useQueryClient();
  const { toast } = useToast();

  const {
    register,
    control,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<TaskFormData>({
    resolver: zodResolver(taskSchema),
    defaultValues: {
      title: "",
      note_id: "none",
    },
  });

  // Query notes to link to task
  const { data: notesData } = useQuery({
    queryKey: ["notes-selection", tenantId],
    queryFn: () => listNotesApi({ page: 1, pageSize: 50 }),
    enabled: open && !!tenantId,
  });

  const mutation = useMutation({
    mutationFn: (data: TaskFormData) =>
      createTaskApi({
        title: data.title,
        note_id: data.note_id && data.note_id !== "none" ? data.note_id : undefined,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tasks"] });
      queryClient.invalidateQueries({ queryKey: ["tasks-count"] });
      toast({
        title: "Tạo công việc thành công",
        description: "Công việc mới đã được tạo với snapshot ghi chú liên kết (nếu có).",
      });
      reset();
      onOpenChange(false);
    },
    onError: (err) => {
      toast({
        title: "Tạo công việc thất bại",
        description: getLocalizedErrorMessage(err),
        variant: "destructive",
      });
    },
  });

  const onSubmit = async (data: TaskFormData) => {
    await mutation.mutateAsync(data);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[480px]">
        <DialogHeader>
          <DialogTitle>Tạo công việc mới</DialogTitle>
          <DialogDescription>
            Tạo một công việc và liên kết ảnh chụp (snapshot) của ghi chú nếu cần.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4 py-2">
          <div className="space-y-2">
            <Label htmlFor="title">Tiêu đề công việc *</Label>
            <Input
              id="title"
              placeholder="VD: Kiểm tra tính năng xác thực"
              disabled={isSubmitting}
              {...register("title")}
            />
            {errors.title && (
              <p className="text-xs text-destructive">{errors.title.message}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="note_id">Ghi chú liên kết (Tuỳ chọn)</Label>
            <Controller
              name="note_id"
              control={control}
              render={({ field }) => (
                <Select
                  value={field.value || "none"}
                  onValueChange={field.onChange}
                  disabled={isSubmitting}
                >
                  <SelectTrigger id="note_id">
                    <SelectValue placeholder="Chọn ghi chú..." />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="none">-- Không liên kết ghi chú --</SelectItem>
                    {notesData?.notes.map((n) => (
                      <SelectItem key={n.id} value={n.id}>
                        {n.title} ({n.status})
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            />
          </div>

          {/* Snapshot Notice Requirement */}
          <div className="rounded-lg border border-blue-500/20 bg-blue-500/10 p-3 text-xs text-blue-800 dark:text-blue-300 space-y-1">
            <div className="flex items-center gap-1.5 font-semibold">
              <Camera className="h-3.5 w-3.5" />
              <span>Cơ chế sao chép Snapshot</span>
            </div>
            <p className="flex items-start gap-1 leading-relaxed">
              <Info className="h-3 w-3 shrink-0 mt-0.5" />
              Hệ thống sẽ lưu bản sao (snapshot) tiêu đề ghi chú tại thời điểm tạo. Nếu ghi
              chú gốc bị chỉnh sửa hoặc xoá sau này, tiêu đề trên công việc này vẫn được bảo
              toàn nguyên vẹn.
            </p>
          </div>

          <DialogFooter className="pt-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isSubmitting}
            >
              Huỷ
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? "Đang tạo..." : "Tạo công việc"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
};
