import React from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { createNoteApi } from "@/api/notes";
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
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

const noteSchema = z.object({
  title: z.string().min(1, "Tiêu đề không được để trống"),
  body: z.string().optional(),
});

type NoteFormData = z.infer<typeof noteSchema>;

interface CreateNoteDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export const CreateNoteDialog: React.FC<CreateNoteDialogProps> = ({
  open,
  onOpenChange,
}) => {
  const queryClient = useQueryClient();
  const { toast } = useToast();

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<NoteFormData>({
    resolver: zodResolver(noteSchema),
    defaultValues: {
      title: "",
      body: "",
    },
  });

  const mutation = useMutation({
    mutationFn: (data: NoteFormData) =>
      createNoteApi({
        title: data.title,
        body: data.body,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notes"] });
      toast({
        title: "Tạo ghi chú thành công",
        description: "Ghi chú mới đã được lưu vào hệ thống.",
      });
      reset();
      onOpenChange(false);
    },
    onError: (err) => {
      toast({
        title: "Tạo ghi chú thất bại",
        description: getLocalizedErrorMessage(err),
        variant: "destructive",
      });
    },
  });

  const onSubmit = async (data: NoteFormData) => {
    await mutation.mutateAsync(data);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[480px]">
        <DialogHeader>
          <DialogTitle>Tạo ghi chú mới</DialogTitle>
          <DialogDescription>
            Điền thông tin bên dưới để tạo ghi chú thuộc tenant hiện tại.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4 py-2">
          <div className="space-y-2">
            <Label htmlFor="title">Tiêu đề *</Label>
            <Input
              id="title"
              placeholder="VD: Kế hoạch triển khai quý 3"
              disabled={isSubmitting}
              {...register("title")}
            />
            {errors.title && (
              <p className="text-xs text-destructive">{errors.title.message}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="body">Nội dung ghi chú</Label>
            <textarea
              id="body"
              rows={4}
              placeholder="Chi tiết nội dung..."
              className="flex w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
              disabled={isSubmitting}
              {...register("body")}
            />
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
              {isSubmitting ? "Đang lưu..." : "Tạo ghi chú"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
};
