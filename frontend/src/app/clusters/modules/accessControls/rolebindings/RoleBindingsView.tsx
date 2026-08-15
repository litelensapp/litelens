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
import { useGetRoleBindings } from "./hooks/data-access/useGetRoleBindings";
import { useDeleteRoleBinding } from "./hooks/data-mutation/useDeleteRoleBinding";
import { useDeleteRoleBindings } from "./hooks/data-mutation/useDeleteRoleBindings";
import { useMainLayoutContext } from "../../../MainLayoutContext";
import { useDetailDrawerContext } from "../../../shared/components/details/DetailDrawerContext";
import { useUnifiedTray } from "../../../shared/components/trays/unified/UnifiedTrayContext";
import { RoleBindingDeleteConfirmationModal } from "./components/RoleBindingDeleteConfirmationModal";

const RoleBindingTableCtaButtons: FC<{ namespace: string; name: string }> = ({
  namespace,
  name,
}) => {
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const { openTab } = useUnifiedTray();
  const { mutate: deleteRoleBinding, isPending: isDeletePending } = useDeleteRoleBinding();

  const handleDeleteConfirm = () => {
    deleteRoleBinding({ namespace, name }, { onSuccess: () => setShowDeleteModal(false) });
  };

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger
          aria-label="Actions"
          className="hover:bg-accent flex size-6 cursor-pointer items-center justify-center rounded-sm"
          onClick={(e) => e.stopPropagation()}
        >
          <MoreVerticalIcon className="size-3.5" />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <ResourceModificationButton
            onClick={() =>
              openTab("modification", { kind: "RoleBinding", name: name, namespace: namespace })
            }
          />
          <ResourceDeletionButton
            disabled={isDeletePending}
            onClick={() => setShowDeleteModal(true)}
          />
        </DropdownMenuContent>
      </DropdownMenu>

      <RoleBindingDeleteConfirmationModal
        open={showDeleteModal}
        mode="single"
        name={name}
        namespace={namespace}
        isPending={isDeletePending}
        onClose={() => setShowDeleteModal(false)}
        onConfirm={handleDeleteConfirm}
      />
    </>
  );
};

export const RoleBindingsView: FC = () => {
  const [search, setSearch] = useState("");
  const [selectedRoleBindingIds, setSelectedRoleBindingIds] = useState<Set<string>>(new Set());
  const [showBulkDeleteModal, setShowBulkDeleteModal] = useState(false);

  const { activeContext, namespaces } = useMainLayoutContext();
  const { onToggleNamespaceDetail, onToggleRoleDetail, onToggleRoleBindingDetail } =
    useDetailDrawerContext();

  const { mutate: deleteRoleBindings, isPending: isBulkDeletePending } = useDeleteRoleBindings();

  const { data: raw = [], isLoading } = useGetRoleBindings({ context: activeContext, namespaces });

  const roleBindings = raw
    .filter((rb) => !search || rb.Name.toLowerCase().includes(search.toLowerCase()))
    .toSorted((a, b) => a.Name.localeCompare(b.Name));

  return (
    <div className="flex h-full flex-col gap-3">
      <div className="flex items-center gap-3">
        <span className="text-h1">Role Bindings</span>
        <span className="text-muted-foreground text-xs">
          {roleBindings.length} item{roleBindings.length !== 1 ? "s" : ""}
        </span>
        <div className="ml-auto flex items-center gap-4">
          <ResourceBulkDeletionButton
            count={selectedRoleBindingIds.size}
            ariaLabel="Delete selected role bindings"
            tooltip="Delete selected RoleBindings"
            onClick={() => setShowBulkDeleteModal(true)}
          />
          <SearchInput
            placeholder="Search Role Bindings..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            wrapperClassName="w-68"
          />
        </div>
      </div>

      <Table containerClassName="flex-1 overflow-y-auto">
        <TableHeader className="bg-background z-sticky sticky top-0">
          <TableRow>
            <TableHead className="w-12">
              <Checkbox
                checked={
                  roleBindings.length > 0 &&
                  roleBindings.every((rb) =>
                    selectedRoleBindingIds.has(`${rb.Namespace}/${rb.Name}`)
                  )
                }
                indeterminate={
                  roleBindings.some((rb) =>
                    selectedRoleBindingIds.has(`${rb.Namespace}/${rb.Name}`)
                  ) &&
                  !roleBindings.every((rb) =>
                    selectedRoleBindingIds.has(`${rb.Namespace}/${rb.Name}`)
                  )
                }
                onCheckedChange={(checked) => {
                  if (checked) {
                    const newSelection = new Set(selectedRoleBindingIds);
                    roleBindings.forEach((rb) => newSelection.add(`${rb.Namespace}/${rb.Name}`));
                    setSelectedRoleBindingIds(newSelection);
                  } else {
                    const newSelection = new Set(selectedRoleBindingIds);
                    roleBindings.forEach((rb) => newSelection.delete(`${rb.Namespace}/${rb.Name}`));
                    setSelectedRoleBindingIds(newSelection);
                  }
                }}
                aria-label="Select all visible role bindings"
              />
            </TableHead>
            <TableHead>Name</TableHead>
            {namespaces.length !== 1 && <TableHead>Namespace</TableHead>}
            <TableHead>Role</TableHead>
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
              columns={namespaces.length !== 1 ? 6 : 5}
              includeCheckbox={true}
              columnWidths={["w-[65%]", "w-[55%]", "w-[45%]", "w-[35%]", "w-[45%]", "w-[30%]"]}
            />
          ) : roleBindings.length === 0 ? (
            <TableRow>
              <TableCell colSpan={namespaces.length !== 1 ? 8 : 7} className="px-0 py-0">
                <EmptyState
                  icon={<Link2Icon className="size-8" />}
                  title="No RoleBindings"
                  description="Create a RoleBinding to grant namespace permissions"
                />
              </TableCell>
            </TableRow>
          ) : (
            roleBindings.map((rb) => {
              const rbId = `${rb.Namespace}/${rb.Name}`;
              const isSelected = selectedRoleBindingIds.has(rbId);
              return (
                <TableRow
                  key={rbId}
                  className={cn(isSelected && "bg-accent/30", "cursor-pointer")}
                  onClick={() => onToggleRoleBindingDetail(rb.Namespace, rb.Name)}
                >
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <Checkbox
                      checked={isSelected}
                      onCheckedChange={() => {
                        const newSelection = new Set(selectedRoleBindingIds);
                        if (isSelected) newSelection.delete(rbId);
                        else newSelection.add(rbId);
                        setSelectedRoleBindingIds(newSelection);
                      }}
                      aria-label={`Select role binding ${rb.Name}`}
                    />
                  </TableCell>
                  <TableCell className="font-mono text-xs">{rb.Name}</TableCell>
                  {namespaces.length !== 1 && (
                    <TableCell className="font-mono text-xs">
                      <ResourceLink
                        onClick={(e) => {
                          e.stopPropagation();
                          onToggleNamespaceDetail(rb.Namespace);
                        }}
                      >
                        {rb.Namespace}
                      </ResourceLink>
                    </TableCell>
                  )}
                  <TableCell className="font-mono text-xs">
                    <ResourceLink
                      truncate
                      onClick={(e) => {
                        e.stopPropagation();
                        onToggleRoleDetail(rb.Namespace, rb.RoleRefName);
                      }}
                    >
                      {rb.RoleRefName}
                    </ResourceLink>
                  </TableCell>
                  <TableCell className="text-xs">{rb.Types || "—"}</TableCell>
                  <TableCell className="max-w-60 font-mono text-xs">
                    <TruncatedText text={rb.Bindings} />
                  </TableCell>
                  <TableCell className="text-xs">{rb.Age}</TableCell>
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <RoleBindingTableCtaButtons namespace={rb.Namespace} name={rb.Name} />
                  </TableCell>
                </TableRow>
              );
            })
          )}
        </TableBody>
      </Table>

      {selectedRoleBindingIds.size > 0 && (
        <RoleBindingDeleteConfirmationModal
          open={showBulkDeleteModal}
          mode="bulk"
          items={Array.from(selectedRoleBindingIds).map((id) => {
            const [ns, name] = id.split("/");
            return { namespace: ns, name };
          })}
          isPending={isBulkDeletePending}
          onClose={() => setShowBulkDeleteModal(false)}
          onConfirm={() => {
            const items = Array.from(selectedRoleBindingIds).map((id) => {
              const [ns, name] = id.split("/");
              return { namespace: ns, name };
            });
            deleteRoleBindings(
              { items },
              {
                onSuccess: () => {
                  setShowBulkDeleteModal(false);
                  setSelectedRoleBindingIds(new Set());
                },
              }
            );
          }}
        />
      )}
    </div>
  );
};
