import {
  Checkbox,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
  EmptyState,
  Link2Icon,
  MoreVerticalIcon,
  ResourceBulkDeletionButton,
  ResourceDeletionButton,
  ResourceLink,
  ResourceModificationButton,
  SearchInput,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  TableSkeletonLoader,
  TruncatedText,
  cn,
} from "@litelens/design-system";
import { FC, useState } from "react";
import type { ClusterRoleBinding } from "./api/resources";
import { useGetClusterRoleBindings } from "./hooks/data-access/useGetClusterRoleBindings";
import { useDeleteClusterRoleBinding } from "./hooks/data-mutation/useDeleteClusterRoleBinding";
import { useDeleteClusterRoleBindings } from "./hooks/data-mutation/useDeleteClusterRoleBindings";
import { useMainLayoutContext } from "../../../MainLayoutContext";
import { useDetailDrawerContext } from "../../../shared/components/details/DetailDrawerContext";
import { useUnifiedTray } from "../../../shared/components/trays/unified/UnifiedTrayContext";
import { ClusterRoleBindingDeleteConfirmationModal } from "./components/ClusterRoleBindingDeleteConfirmationModal";

interface ClusterRoleBindingTableCtaButtonsProps {
  name: string;
}

const ClusterRoleBindingTableCtaButtons: FC<ClusterRoleBindingTableCtaButtonsProps> = ({
  name,
}) => {
  const { openTab } = useUnifiedTray();

  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const { mutate: deleteClusterRoleBinding, isPending: isDeletePending } =
    useDeleteClusterRoleBinding();

  const handleDeleteConfirm = () => {
    deleteClusterRoleBinding({ name }, { onSuccess: () => setShowDeleteModal(false) });
  };

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger
          aria-label="Actions"
          className="flex size-6 cursor-pointer items-center justify-center rounded-sm hover:bg-accent"
          onClick={(e) => e.stopPropagation()}
        >
          <MoreVerticalIcon className="size-3.5" />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <ResourceModificationButton
            onClick={() => openTab("modification", { kind: "ClusterRoleBinding", name: name })}
          />
          <ResourceDeletionButton
            disabled={isDeletePending}
            onClick={() => setShowDeleteModal(true)}
          />
        </DropdownMenuContent>
      </DropdownMenu>

      <ClusterRoleBindingDeleteConfirmationModal
        open={showDeleteModal}
        mode="single"
        name={name}
        isPending={isDeletePending}
        onClose={() => setShowDeleteModal(false)}
        onConfirm={handleDeleteConfirm}
      />
    </>
  );
};

export const ClusterRoleBindingsView: FC = () => {
  const [search, setSearch] = useState("");
  const [selectedClusterRoleBindingNames, setSelectedClusterRoleBindingNames] = useState<
    Set<string>
  >(new Set());
  const [showBulkDeleteModal, setShowBulkDeleteModal] = useState(false);

  const { activeContext } = useMainLayoutContext();
  const { onToggleClusterRoleDetail, onToggleClusterRoleBindingDetail } = useDetailDrawerContext();

  const { mutate: deleteClusterRoleBindings, isPending: isBulkDeletePending } =
    useDeleteClusterRoleBindings();

  const { data: raw = [], isLoading } = useGetClusterRoleBindings(activeContext);

  const clusterRoleBindings = raw
    .filter((crb) => !search || crb.Name.toLowerCase().includes(search.toLowerCase()))
    .toSorted((a, b) => a.Name.localeCompare(b.Name));

  const handleRowClick = (crb: ClusterRoleBinding) => {
    onToggleClusterRoleBindingDetail(crb.Name);
  };

  return (
    <div className="flex h-full flex-col gap-3">
      <div className="flex items-center gap-3">
        <span className="text-h1">Cluster Role Bindings</span>
        <span className="text-xs text-muted-foreground">
          {clusterRoleBindings.length} item{clusterRoleBindings.length !== 1 ? "s" : ""}
        </span>
        <div className="ml-auto flex items-center gap-4">
          <ResourceBulkDeletionButton
            count={selectedClusterRoleBindingNames.size}
            ariaLabel="Delete selected cluster role bindings"
            tooltip="Delete selected ClusterRoleBindings"
            onClick={() => setShowBulkDeleteModal(true)}
          />
          <SearchInput
            placeholder="Search Cluster Role Bindings..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            wrapperClassName="w-68"
          />
        </div>
      </div>

      <Table containerClassName="flex-1 overflow-y-auto">
        <TableHeader className="z-sticky sticky top-0 bg-background">
          <TableRow>
            <TableHead className="w-12">
              <Checkbox
                checked={
                  clusterRoleBindings.length > 0 &&
                  clusterRoleBindings.every((crb) => selectedClusterRoleBindingNames.has(crb.Name))
                }
                indeterminate={
                  clusterRoleBindings.some((crb) =>
                    selectedClusterRoleBindingNames.has(crb.Name)
                  ) &&
                  !clusterRoleBindings.every((crb) => selectedClusterRoleBindingNames.has(crb.Name))
                }
                onCheckedChange={(checked) => {
                  if (checked) {
                    const newSelection = new Set(selectedClusterRoleBindingNames);
                    clusterRoleBindings.forEach((crb) => newSelection.add(crb.Name));
                    setSelectedClusterRoleBindingNames(newSelection);
                  } else {
                    const newSelection = new Set(selectedClusterRoleBindingNames);
                    clusterRoleBindings.forEach((crb) => newSelection.delete(crb.Name));
                    setSelectedClusterRoleBindingNames(newSelection);
                  }
                }}
                aria-label="Select all visible cluster role bindings"
              />
            </TableHead>
            <TableHead>Name</TableHead>
            <TableHead>Cluster Role</TableHead>
            <TableHead>Types</TableHead>
            <TableHead>Bindings</TableHead>
            <TableHead>Age</TableHead>
            <TableHead className="w-8" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            <TableSkeletonLoader
              rows={5}
              columns={5}
              includeCheckbox={true}
              columnWidths={["w-[65%]", "w-[55%]", "w-[35%]", "w-[40%]", "w-[30%]"]}
            />
          ) : clusterRoleBindings.length === 0 ? (
            <TableRow>
              <TableCell colSpan={7} className="px-0 py-0">
                <EmptyState
                  icon={<Link2Icon className="size-8" />}
                  title="No ClusterRoleBindings"
                  description="Create a ClusterRoleBinding to grant cluster-wide permissions"
                />
              </TableCell>
            </TableRow>
          ) : (
            clusterRoleBindings.map((crb) => {
              const isSelected = selectedClusterRoleBindingNames.has(crb.Name);
              const types = [...new Set((crb.Subjects ?? []).map((s) => s.Kind))].join(", ");
              return (
                <TableRow
                  key={crb.Name}
                  className={cn(isSelected && "bg-accent/30", "cursor-pointer")}
                  onClick={() => handleRowClick(crb)}
                >
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <Checkbox
                      checked={isSelected}
                      onCheckedChange={() => {
                        const newSelection = new Set(selectedClusterRoleBindingNames);
                        if (isSelected) newSelection.delete(crb.Name);
                        else newSelection.add(crb.Name);
                        setSelectedClusterRoleBindingNames(newSelection);
                      }}
                      aria-label={`Select cluster role binding ${crb.Name}`}
                    />
                  </TableCell>
                  <TableCell className="font-mono text-xs">{crb.Name}</TableCell>
                  <TableCell className="font-mono text-xs">
                    <ResourceLink
                      truncate
                      onClick={(e) => {
                        e.stopPropagation();
                        onToggleClusterRoleDetail(crb.RoleRefName);
                      }}
                    >
                      {crb.RoleRefName}
                    </ResourceLink>
                  </TableCell>
                  <TableCell className="text-xs">{types || "—"}</TableCell>
                  <TableCell className="max-w-60 font-mono text-xs">
                    <TruncatedText text={crb.Bindings} />
                  </TableCell>
                  <TableCell className="text-xs">{crb.Age}</TableCell>
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <ClusterRoleBindingTableCtaButtons name={crb.Name} />
                  </TableCell>
                </TableRow>
              );
            })
          )}
        </TableBody>
      </Table>

      {selectedClusterRoleBindingNames.size > 0 && (
        <ClusterRoleBindingDeleteConfirmationModal
          open={showBulkDeleteModal}
          mode="bulk"
          items={Array.from(selectedClusterRoleBindingNames).map((name) => ({ name }))}
          isPending={isBulkDeletePending}
          onClose={() => setShowBulkDeleteModal(false)}
          onConfirm={() => {
            const items = Array.from(selectedClusterRoleBindingNames).map((name) => ({ name }));
            deleteClusterRoleBindings(
              { items },
              {
                onSuccess: () => {
                  setShowBulkDeleteModal(false);
                  setSelectedClusterRoleBindingNames(new Set());
                },
              }
            );
          }}
        />
      )}
    </div>
  );
};
