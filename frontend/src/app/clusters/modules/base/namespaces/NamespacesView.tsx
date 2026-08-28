import {
  AnnotationBadge,
  BoxesIcon,
  Button,
  Checkbox,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
  EmptyState,
  MoreVerticalIcon,
  ResourceBulkDeletionButton,
  ResourceCreationButton,
  ResourceDeletionButton,
  ResourceModificationButton,
  SearchInput,
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
import { useMainLayoutContext } from "../../../MainLayoutContext";
import { useDetailDrawerContext } from "../../../shared/components/details/DetailDrawerContext";
import { useUnifiedTray } from "../../../shared/components/trays/unified/UnifiedTrayContext";
import { NamespaceCreationModal } from "./components/NamespaceCreationModal";
import { NamespaceDeleteConfirmationModal } from "./components/NamespaceDeleteConfirmationModal";
import { NamespaceStatusBadge } from "./components/NamespaceStatusBadge";
import { useGetNamespaces } from "./hooks/data-access/useGetNamespaces";
import { useDeleteNamespace } from "./hooks/data-mutation/useDeleteNamespace";
import { useDeleteNamespaces } from "./hooks/data-mutation/useDeleteNamespaces";

const NamespaceTableCtaButtons: FC<{ name: string }> = ({ name }) => {
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const { mutate: deleteNamespace, isPending: isDeletePending } = useDeleteNamespace();
  const { openTab } = useUnifiedTray();

  const handleDeleteConfirm = () => {
    deleteNamespace({ name }, { onSuccess: () => setShowDeleteModal(false) });
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
            onClick={() => openTab("modification", { kind: "Namespace", name })}
          />
          <ResourceDeletionButton
            disabled={isDeletePending}
            onClick={() => setShowDeleteModal(true)}
          />
        </DropdownMenuContent>
      </DropdownMenu>

      <NamespaceDeleteConfirmationModal
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

export const NamespacesView: FC = () => {
  const [search, setSearch] = useState("");
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [selectedNamespaceNames, setSelectedNamespaceNames] = useState<Set<string>>(new Set());
  const [showBulkDeleteModal, setShowBulkDeleteModal] = useState(false);

  const { activeContext } = useMainLayoutContext();
  const { onToggleNamespaceDetail } = useDetailDrawerContext();

  const { mutate: deleteNamespaces, isPending: isBulkDeletePending } = useDeleteNamespaces();

  const { data: raw = [], isLoading } = useGetNamespaces(activeContext);

  const namespaces = raw
    .filter((ns) => !search || ns.Name.toLowerCase().includes(search.toLowerCase()))
    .toSorted((a, b) => a.Name.localeCompare(b.Name));

  return (
    <div className="flex h-full flex-col gap-3">
      <div className="flex items-center gap-3">
        <span className="text-h1">Namespaces</span>
        <span className="text-xs text-muted-foreground">
          {namespaces.length} item{namespaces.length === 1 ? "" : "s"}
        </span>
        <div className="ml-auto flex items-center gap-4">
          <ResourceBulkDeletionButton
            count={selectedNamespaceNames.size}
            ariaLabel="Delete selected namespaces"
            tooltip="Delete selected Namespaces"
            onClick={() => setShowBulkDeleteModal(true)}
          />
          <ResourceCreationButton
            ariaLabel="Create Namespace"
            tooltip="Create Namespace"
            onClick={() => setIsCreateOpen(true)}
          />
          <SearchInput
            placeholder="Search Namespaces..."
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
                  namespaces.length > 0 &&
                  namespaces.every((ns) => selectedNamespaceNames.has(ns.Name))
                }
                indeterminate={
                  namespaces.some((ns) => selectedNamespaceNames.has(ns.Name)) &&
                  !namespaces.every((ns) => selectedNamespaceNames.has(ns.Name))
                }
                onCheckedChange={(checked) => {
                  if (checked) {
                    const newSelection = new Set(selectedNamespaceNames);
                    namespaces.forEach((ns) => newSelection.add(ns.Name));
                    setSelectedNamespaceNames(newSelection);
                  } else {
                    const newSelection = new Set(selectedNamespaceNames);
                    namespaces.forEach((ns) => newSelection.delete(ns.Name));
                    setSelectedNamespaceNames(newSelection);
                  }
                }}
                aria-label="Select all visible namespaces"
              />
            </TableHead>
            <TableHead>Name</TableHead>
            <TableHead>Labels</TableHead>
            <TableHead>Age</TableHead>
            <TableHead>Status</TableHead>
            <TableHead className="w-8" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            <TableSkeletonLoader
              rows={5}
              columns={4}
              includeCheckbox={true}
              columnWidths={["w-[65%]", "w-[40%]", "w-[30%]", "w-[40%]"]}
            />
          ) : namespaces.length === 0 ? (
            <TableRow>
              <TableCell colSpan={6} className="px-0 py-0">
                <EmptyState
                  icon={<BoxesIcon className="size-8" />}
                  title="No Namespaces"
                  description="Create a namespace to organize resources"
                  action={
                    <Button variant="default" size="default" onClick={() => setIsCreateOpen(true)}>
                      Create Namespace
                    </Button>
                  }
                />
              </TableCell>
            </TableRow>
          ) : (
            namespaces.map((ns) => {
              const isSelected = selectedNamespaceNames.has(ns.Name);
              return (
                <TableRow
                  key={ns.Name}
                  className={cn(isSelected && "bg-accent/30")}
                  onClick={() => onToggleNamespaceDetail(ns.Name)}
                >
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <Checkbox
                      checked={isSelected}
                      onCheckedChange={() => {
                        const newSelection = new Set(selectedNamespaceNames);
                        if (isSelected) newSelection.delete(ns.Name);
                        else newSelection.add(ns.Name);
                        setSelectedNamespaceNames(newSelection);
                      }}
                      aria-label={`Select namespace ${ns.Name}`}
                    />
                  </TableCell>
                  <TableCell className="font-mono text-xs">{ns.Name}</TableCell>
                  <TableCell>
                    <div className="flex flex-wrap gap-1">
                      {Object.keys(ns.Labels ?? {}).length > 0 ? (
                        Object.entries(ns.Labels ?? {}).map(([k, v]) => (
                          <AnnotationBadge key={k} label={`${k}=${v}`} />
                        ))
                      ) : (
                        <span className="text-xs text-muted-foreground">—</span>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>{ns.Age}</TableCell>
                  <TableCell>
                    <NamespaceStatusBadge status={ns.Status} />
                  </TableCell>
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <NamespaceTableCtaButtons name={ns.Name} />
                  </TableCell>
                </TableRow>
              );
            })
          )}
        </TableBody>
      </Table>

      {selectedNamespaceNames.size > 0 && (
        <NamespaceDeleteConfirmationModal
          open={showBulkDeleteModal}
          mode="bulk"
          items={Array.from(selectedNamespaceNames)}
          isPending={isBulkDeletePending}
          onClose={() => setShowBulkDeleteModal(false)}
          onConfirm={() => {
            deleteNamespaces(
              { names: Array.from(selectedNamespaceNames) },
              {
                onSuccess: () => {
                  setShowBulkDeleteModal(false);
                  setSelectedNamespaceNames(new Set());
                },
              }
            );
          }}
        />
      )}

      <NamespaceCreationModal open={isCreateOpen} onClose={() => setIsCreateOpen(false)} />
    </div>
  );
};
