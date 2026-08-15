import {
  Checkbox,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
  EmptyState,
  HardDriveIcon,
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
  cn,
} from "@litelens/design-system";
import { FC, useState } from "react";
import { useGetPersistentVolumeClaims } from "./hooks/data-access/useGetPersistentVolumeClaims";
import { useDeletePersistentVolumeClaim } from "./hooks/data-mutation/useDeletePersistentVolumeClaim";
import { useDeletePersistentVolumeClaims } from "./hooks/data-mutation/useDeletePersistentVolumeClaims";
import { useMainLayoutContext } from "../../../MainLayoutContext";
import { useDetailDrawerContext } from "../../../shared/components/details/DetailDrawerContext";
import { useUnifiedTray } from "../../../shared/components/trays/unified/UnifiedTrayContext";
import { PersistentVolumeClaimDeleteConfirmationModal } from "./components/PersistentVolumeClaimDeleteConfirmationModal";
import { PersistentVolumeClaimStatusBadge } from "./components/PersistentVolumeClaimStatusBadge";

interface PersistentVolumeClaimTableCtaButtonsProps {
  name: string;
  namespace: string;
}

const PersistentVolumeClaimTableCtaButtons: FC<PersistentVolumeClaimTableCtaButtonsProps> = ({
  name,
  namespace,
}) => {
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const { openTab } = useUnifiedTray();
  const { mutate: deletePersistentVolumeClaim, isPending: isDeletePending } =
    useDeletePersistentVolumeClaim();

  const handleDeleteConfirm = () => {
    deletePersistentVolumeClaim(
      { namespace, name },
      { onSuccess: () => setShowDeleteModal(false) }
    );
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
              openTab("modification", { kind: "PersistentVolumeClaim", name, namespace })
            }
          />
          <ResourceDeletionButton
            disabled={isDeletePending}
            onClick={() => setShowDeleteModal(true)}
          />
        </DropdownMenuContent>
      </DropdownMenu>

      <PersistentVolumeClaimDeleteConfirmationModal
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

export const PersistentVolumeClaimsView: FC = () => {
  const [search, setSearch] = useState("");
  const [selectedPVCKeys, setSelectedPVCKeys] = useState<Set<string>>(new Set());
  const [showBulkDeleteModal, setShowBulkDeleteModal] = useState(false);

  const { activeContext, namespaces } = useMainLayoutContext();
  const { onToggleNamespaceDetail, onTogglePersistentVolumeClaimDetail } = useDetailDrawerContext();

  const { mutate: deletePersistentVolumeClaims, isPending: isBulkDeletePending } =
    useDeletePersistentVolumeClaims();

  const { data: raw = [], isLoading } = useGetPersistentVolumeClaims({
    context: activeContext,
    namespaces,
  });

  const pvcs = raw
    .filter((p) => !search || p.Name.toLowerCase().includes(search.toLowerCase()))
    .toSorted((a, b) => a.Name.localeCompare(b.Name));

  return (
    <div className="flex h-full flex-col gap-3">
      <div className="flex items-center gap-3">
        <span className="text-h1">Persistent Volume Claims</span>
        <span className="text-muted-foreground text-xs">
          {pvcs.length} item{pvcs.length !== 1 ? "s" : ""}
        </span>
        <div className="ml-auto flex items-center gap-4">
          <ResourceBulkDeletionButton
            count={selectedPVCKeys.size}
            ariaLabel="Delete selected persistent volume claims"
            tooltip="Delete selected Persistent Volume Claims"
            onClick={() => setShowBulkDeleteModal(true)}
          />
          <SearchInput
            placeholder="Search Persistent Volume Claims..."
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
                  pvcs.length > 0 &&
                  pvcs.every((pvc) => selectedPVCKeys.has(`${pvc.Namespace}/${pvc.Name}`))
                }
                indeterminate={
                  pvcs.some((pvc) => selectedPVCKeys.has(`${pvc.Namespace}/${pvc.Name}`)) &&
                  !pvcs.every((pvc) => selectedPVCKeys.has(`${pvc.Namespace}/${pvc.Name}`))
                }
                onCheckedChange={(checked) => {
                  if (checked) {
                    const newSelection = new Set(selectedPVCKeys);
                    pvcs.forEach((pvc) => newSelection.add(`${pvc.Namespace}/${pvc.Name}`));
                    setSelectedPVCKeys(newSelection);
                  } else {
                    const newSelection = new Set(selectedPVCKeys);
                    pvcs.forEach((pvc) => newSelection.delete(`${pvc.Namespace}/${pvc.Name}`));
                    setSelectedPVCKeys(newSelection);
                  }
                }}
                aria-label="Select all visible persistent volume claims"
              />
            </TableHead>
            <TableHead>Name</TableHead>
            {namespaces.length !== 1 && <TableHead>Namespace</TableHead>}
            <TableHead>Storage Class</TableHead>
            <TableHead>Size</TableHead>
            <TableHead>Pods</TableHead>
            <TableHead>Age</TableHead>
            <TableHead>Status</TableHead>
            <TableHead className="w-8" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            <TableSkeletonLoader
              rows={5}
              columns={namespaces.length !== 1 ? 6 : 5}
              includeCheckbox={true}
              columnWidths={["w-[65%]", "w-[55%]", "w-[35%]", "w-[30%]", "w-[45%]", "w-[30%]"]}
            />
          ) : pvcs.length === 0 ? (
            <TableRow>
              <TableCell colSpan={namespaces.length !== 1 ? 9 : 8} className="px-0 py-0">
                <EmptyState
                  icon={<HardDriveIcon className="size-8" />}
                  title="No PersistentVolumeClaims"
                  description="Create a PersistentVolumeClaim to request storage"
                />
              </TableCell>
            </TableRow>
          ) : (
            pvcs.map((p) => {
              const key = `${p.Namespace}/${p.Name}`;
              const isSelected = selectedPVCKeys.has(key);
              return (
                <TableRow
                  key={key}
                  className={cn(isSelected && "bg-accent/30")}
                  onClick={() => onTogglePersistentVolumeClaimDetail(p.Namespace, p.Name)}
                >
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <Checkbox
                      checked={isSelected}
                      onCheckedChange={() => {
                        const newSelection = new Set(selectedPVCKeys);
                        if (isSelected) newSelection.delete(key);
                        else newSelection.add(key);
                        setSelectedPVCKeys(newSelection);
                      }}
                      aria-label={`Select persistent volume claim ${p.Namespace}/${p.Name}`}
                    />
                  </TableCell>
                  <TableCell className="font-mono text-xs">{p.Name}</TableCell>
                  {namespaces.length !== 1 && (
                    <TableCell className="text-xs">
                      <ResourceLink
                        onClick={(e) => {
                          e.stopPropagation();
                          onToggleNamespaceDetail(p.Namespace);
                        }}
                      >
                        {p.Namespace}
                      </ResourceLink>
                    </TableCell>
                  )}
                  <TableCell className="text-xs">{p.StorageClass}</TableCell>
                  <TableCell className="font-mono text-xs">{p.Size}</TableCell>
                  <TableCell className="max-w-xs truncate text-xs">{p.Pods}</TableCell>
                  <TableCell className="text-xs">{p.Age}</TableCell>
                  <TableCell>
                    <PersistentVolumeClaimStatusBadge status={p.Status} />
                  </TableCell>
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <PersistentVolumeClaimTableCtaButtons name={p.Name} namespace={p.Namespace} />
                  </TableCell>
                </TableRow>
              );
            })
          )}
        </TableBody>
      </Table>

      {selectedPVCKeys.size > 0 && (
        <PersistentVolumeClaimDeleteConfirmationModal
          open={showBulkDeleteModal}
          mode="bulk"
          items={Array.from(selectedPVCKeys).map((key) => {
            const [ns, name] = key.split("/");
            return { namespace: ns, name };
          })}
          isPending={isBulkDeletePending}
          onClose={() => setShowBulkDeleteModal(false)}
          onConfirm={() => {
            deletePersistentVolumeClaims(
              {
                items: Array.from(selectedPVCKeys).map((key) => {
                  const [ns, name] = key.split("/");
                  return { namespace: ns, name };
                }),
              },
              {
                onSuccess: () => {
                  setShowBulkDeleteModal(false);
                  setSelectedPVCKeys(new Set());
                },
              }
            );
          }}
        />
      )}
    </div>
  );
};
