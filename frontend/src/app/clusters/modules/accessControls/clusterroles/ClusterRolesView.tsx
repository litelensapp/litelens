import {
  Checkbox,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
  EmptyState,
  MoreVerticalIcon,
  ResourceBulkDeletionButton,
  ResourceDeletionButton,
  ResourceModificationButton,
  SearchInput,
  ShieldIcon,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  TableSkeletonLoader,
  cn,
} from "@litelens/design-system";
import { FC, useState } from "react";
import { useGetClusterRoles } from "./hooks/data-access/useGetClusterRoles";
import { useDeleteClusterRole } from "./hooks/data-mutation/useDeleteClusterRole";
import { useDeleteClusterRoles } from "./hooks/data-mutation/useDeleteClusterRoles";
import { useMainLayoutContext } from "../../../MainLayoutContext";
import { useDetailDrawerContext } from "../../../shared/components/details/DetailDrawerContext";
import { useUnifiedTray } from "../../../shared/components/trays/unified/UnifiedTrayContext";
import { ClusterRoleDeleteConfirmationModal } from "./components/ClusterRoleDeleteConfirmationModal";

interface ClusterRoleTableCtaButtonsProps {
  name: string;
}

const ClusterRoleTableCtaButtons: FC<ClusterRoleTableCtaButtonsProps> = ({ name }) => {
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const { mutate: deleteClusterRole, isPending: isDeletePending } = useDeleteClusterRole();
  const { openTab } = useUnifiedTray();

  const handleDeleteConfirm = () => {
    deleteClusterRole({ name }, { onSuccess: () => setShowDeleteModal(false) });
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
            onClick={() => openTab("modification", { kind: "ClusterRole", name })}
          />
          <ResourceDeletionButton
            disabled={isDeletePending}
            onClick={() => setShowDeleteModal(true)}
          />
        </DropdownMenuContent>
      </DropdownMenu>

      <ClusterRoleDeleteConfirmationModal
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

export const ClusterRolesView: FC = () => {
  const [search, setSearch] = useState("");
  const [selectedClusterRoleNames, setSelectedClusterRoleNames] = useState<Set<string>>(new Set());
  const [showBulkDeleteModal, setShowBulkDeleteModal] = useState(false);

  const { activeContext } = useMainLayoutContext();
  const { onToggleClusterRoleDetail } = useDetailDrawerContext();

  const { mutate: deleteClusterRoles, isPending: isBulkDeletePending } = useDeleteClusterRoles();

  const { data: raw = [], isLoading } = useGetClusterRoles(activeContext);

  const clusterRoles = raw
    .filter((cr) => !search || cr.Name.toLowerCase().includes(search.toLowerCase()))
    .toSorted((a, b) => a.Name.localeCompare(b.Name));

  return (
    <div className="flex h-full flex-col gap-3">
      <div className="flex items-center gap-3">
        <span className="text-h1">Cluster Roles</span>
        <span className="text-xs text-muted-foreground">
          {clusterRoles.length} item{clusterRoles.length !== 1 ? "s" : ""}
        </span>
        <div className="ml-auto flex items-center gap-4">
          <ResourceBulkDeletionButton
            count={selectedClusterRoleNames.size}
            ariaLabel="Delete selected cluster roles"
            tooltip="Delete selected ClusterRoles"
            onClick={() => setShowBulkDeleteModal(true)}
          />
          <SearchInput
            placeholder="Search Cluster Roles..."
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
                  clusterRoles.length > 0 &&
                  clusterRoles.every((cr) => selectedClusterRoleNames.has(cr.Name))
                }
                indeterminate={
                  clusterRoles.some((cr) => selectedClusterRoleNames.has(cr.Name)) &&
                  !clusterRoles.every((cr) => selectedClusterRoleNames.has(cr.Name))
                }
                onCheckedChange={(checked) => {
                  if (checked) {
                    const newSelection = new Set(selectedClusterRoleNames);
                    clusterRoles.forEach((cr) => newSelection.add(cr.Name));
                    setSelectedClusterRoleNames(newSelection);
                  } else {
                    const newSelection = new Set(selectedClusterRoleNames);
                    clusterRoles.forEach((cr) => newSelection.delete(cr.Name));
                    setSelectedClusterRoleNames(newSelection);
                  }
                }}
                aria-label="Select all visible cluster roles"
              />
            </TableHead>
            <TableHead>Name</TableHead>
            <TableHead>Age</TableHead>
            <TableHead className="w-8" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            <TableSkeletonLoader
              rows={5}
              columns={2}
              includeCheckbox={true}
              columnWidths={["w-[65%]", "w-[30%]"]}
            />
          ) : clusterRoles.length === 0 ? (
            <TableRow>
              <TableCell colSpan={4} className="px-0 py-0">
                <EmptyState
                  icon={<ShieldIcon className="size-8" />}
                  title="No ClusterRoles"
                  description="Create a ClusterRole to define cluster-wide permissions"
                />
              </TableCell>
            </TableRow>
          ) : (
            clusterRoles.map((cr) => {
              const isSelected = selectedClusterRoleNames.has(cr.Name);
              return (
                <TableRow
                  key={cr.Name}
                  className={cn(isSelected && "bg-accent/30", "cursor-pointer")}
                  onClick={() => onToggleClusterRoleDetail(cr.Name)}
                >
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <Checkbox
                      checked={isSelected}
                      onCheckedChange={() => {
                        const newSelection = new Set(selectedClusterRoleNames);
                        if (isSelected) newSelection.delete(cr.Name);
                        else newSelection.add(cr.Name);
                        setSelectedClusterRoleNames(newSelection);
                      }}
                      aria-label={`Select cluster role ${cr.Name}`}
                    />
                  </TableCell>
                  <TableCell className="font-mono text-xs">{cr.Name}</TableCell>
                  <TableCell className="text-xs">{cr.Age}</TableCell>
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <ClusterRoleTableCtaButtons name={cr.Name} />
                  </TableCell>
                </TableRow>
              );
            })
          )}
        </TableBody>
      </Table>

      {selectedClusterRoleNames.size > 0 && (
        <ClusterRoleDeleteConfirmationModal
          open={showBulkDeleteModal}
          mode="bulk"
          items={Array.from(selectedClusterRoleNames).map((name) => ({ name }))}
          isPending={isBulkDeletePending}
          onClose={() => setShowBulkDeleteModal(false)}
          onConfirm={() => {
            const items = Array.from(selectedClusterRoleNames).map((name) => ({ name }));
            deleteClusterRoles(
              { items },
              {
                onSuccess: () => {
                  setShowBulkDeleteModal(false);
                  setSelectedClusterRoleNames(new Set());
                },
              }
            );
          }}
        />
      )}
    </div>
  );
};
