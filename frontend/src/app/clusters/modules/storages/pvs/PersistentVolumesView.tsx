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
import { useGetPersistentVolumes } from "./hooks/data-access/useGetPersistentVolumes";
import { useDeletePersistentVolume } from "./hooks/data-mutation/useDeletePersistentVolume";
import { useDeletePersistentVolumes } from "./hooks/data-mutation/useDeletePersistentVolumes";
import { useMainLayoutContext } from "../../../MainLayoutContext";
import { useDetailDrawerContext } from "../../../shared/components/details/DetailDrawerContext";
import { useUnifiedTray } from "../../../shared/components/trays/unified/UnifiedTrayContext";
import { PersistentVolumeDeleteConfirmationModal } from "./components/PersistentVolumeDeleteConfirmationModal";
import { PersistentVolumeStatusBadge } from "./components/PersistentVolumeStatusBadge";

interface PersistentVolumeTableCtaButtonsProps {
  name: string;
}

const PersistentVolumeTableCtaButtons: FC<PersistentVolumeTableCtaButtonsProps> = ({ name }) => {
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const { openTab } = useUnifiedTray();
  const { mutate: deletePersistentVolume, isPending: isDeletePending } =
    useDeletePersistentVolume();

  const handleDeleteConfirm = () => {
    deletePersistentVolume({ name }, { onSuccess: () => setShowDeleteModal(false) });
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
            onClick={() => openTab("modification", { kind: "PersistentVolume", name })}
          />
          <ResourceDeletionButton
            disabled={isDeletePending}
            onClick={() => setShowDeleteModal(true)}
          />
        </DropdownMenuContent>
      </DropdownMenu>

      <PersistentVolumeDeleteConfirmationModal
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

export const PersistentVolumesView: FC = () => {
  const { activeContext } = useMainLayoutContext();
  const { onTogglePersistentVolumeDetail } = useDetailDrawerContext();

  const [search, setSearch] = useState("");
  const [selectedPVNames, setSelectedPVNames] = useState<Set<string>>(new Set());
  const [showBulkDeleteModal, setShowBulkDeleteModal] = useState(false);

  const { mutate: deletePersistentVolumes, isPending: isBulkDeletePending } =
    useDeletePersistentVolumes();

  const { data: raw = [], isLoading } = useGetPersistentVolumes(activeContext);

  const pvs = raw
    .filter((p) => !search || p.Name.toLowerCase().includes(search.toLowerCase()))
    .toSorted((a, b) => a.Name.localeCompare(b.Name));

  return (
    <div className="flex h-full flex-col gap-3">
      <div className="flex items-center gap-3">
        <span className="text-h1">Persistent Volumes</span>
        <span className="text-xs text-muted-foreground">
          {pvs.length} item{pvs.length !== 1 ? "s" : ""}
        </span>
        <div className="ml-auto flex items-center gap-4">
          <ResourceBulkDeletionButton
            count={selectedPVNames.size}
            ariaLabel="Delete selected persistent volumes"
            tooltip="Delete selected Persistent Volumes"
            onClick={() => setShowBulkDeleteModal(true)}
          />
          <SearchInput
            placeholder="Search Persistent Volumes..."
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
                checked={pvs.length > 0 && pvs.every((pv) => selectedPVNames.has(pv.Name))}
                indeterminate={
                  pvs.some((pv) => selectedPVNames.has(pv.Name)) &&
                  !pvs.every((pv) => selectedPVNames.has(pv.Name))
                }
                onCheckedChange={(checked) => {
                  if (checked) {
                    const newSelection = new Set(selectedPVNames);
                    pvs.forEach((pv) => newSelection.add(pv.Name));
                    setSelectedPVNames(newSelection);
                  } else {
                    const newSelection = new Set(selectedPVNames);
                    pvs.forEach((pv) => newSelection.delete(pv.Name));
                    setSelectedPVNames(newSelection);
                  }
                }}
                aria-label="Select all visible persistent volumes"
              />
            </TableHead>
            <TableHead>Name</TableHead>
            <TableHead>Storage Class</TableHead>
            <TableHead>Capacity</TableHead>
            <TableHead>Claim</TableHead>
            <TableHead>Age</TableHead>
            <TableHead>Status</TableHead>
            <TableHead className="w-8" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            <TableSkeletonLoader
              rows={5}
              columns={5}
              includeCheckbox={true}
              columnWidths={["w-[65%]", "w-[35%]", "w-[30%]", "w-[45%]", "w-[30%]"]}
            />
          ) : pvs.length === 0 ? (
            <TableRow>
              <TableCell colSpan={8} className="px-0 py-0">
                <EmptyState
                  icon={<HardDriveIcon className="size-8" />}
                  title="No PersistentVolumes"
                  description="PersistentVolumes represent storage provisioned in the cluster"
                />
              </TableCell>
            </TableRow>
          ) : (
            pvs.map((p) => {
              const isSelected = selectedPVNames.has(p.Name);
              return (
                <TableRow
                  key={p.Name}
                  className={cn(isSelected && "bg-accent/30")}
                  onClick={() => onTogglePersistentVolumeDetail(p.Name)}
                >
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <Checkbox
                      checked={isSelected}
                      onCheckedChange={() => {
                        const newSelection = new Set(selectedPVNames);
                        if (isSelected) newSelection.delete(p.Name);
                        else newSelection.add(p.Name);
                        setSelectedPVNames(newSelection);
                      }}
                      aria-label={`Select persistent volume ${p.Name}`}
                    />
                  </TableCell>
                  <TableCell className="font-mono text-xs">{p.Name}</TableCell>
                  <TableCell className="text-xs">{p.StorageClass}</TableCell>
                  <TableCell className="font-mono text-xs">{p.Capacity}</TableCell>
                  <TableCell className="text-xs">
                    <ResourceLink>{p.Claim}</ResourceLink>
                  </TableCell>
                  <TableCell className="text-xs">{p.Age}</TableCell>
                  <TableCell>
                    <PersistentVolumeStatusBadge status={p.Status} />
                  </TableCell>
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <PersistentVolumeTableCtaButtons name={p.Name} />
                  </TableCell>
                </TableRow>
              );
            })
          )}
        </TableBody>
      </Table>

      {selectedPVNames.size > 0 && (
        <PersistentVolumeDeleteConfirmationModal
          open={showBulkDeleteModal}
          mode="bulk"
          items={Array.from(selectedPVNames)}
          isPending={isBulkDeletePending}
          onClose={() => setShowBulkDeleteModal(false)}
          onConfirm={() => {
            deletePersistentVolumes(
              { names: Array.from(selectedPVNames) },
              {
                onSuccess: () => {
                  setShowBulkDeleteModal(false);
                  setSelectedPVNames(new Set());
                },
              }
            );
          }}
        />
      )}
    </div>
  );
};
