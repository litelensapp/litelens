import {
  Checkbox,
  ClockIcon,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
  EmptyState,
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
} from "@litelens/design-system";
import { FC, useMemo, useState } from "react";
import { useGetLeases } from "./hooks/data-access/useGetLeases";
import { useDeleteLease } from "./hooks/data-mutation/useDeleteLease";
import { useDeleteLeases } from "./hooks/data-mutation/useDeleteLeases";
import { useMainLayoutContext } from "../../../MainLayoutContext";
import { useDetailDrawerContext } from "../../../shared/components/details/DetailDrawerContext";
import { useUnifiedTray } from "../../../shared/components/trays/unified/UnifiedTrayContext";
import { LeaseDeleteConfirmationModal } from "./components/LeaseDeleteConfirmationModal";

interface LeaseTableCtaButtonsProps {
  namespace: string;
  name: string;
}

const LeaseTableCtaButtons: FC<LeaseTableCtaButtonsProps> = ({ namespace, name }) => {
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const { openTab } = useUnifiedTray();
  const { mutate: deleteSingle, isPending: isDeletePending } = useDeleteLease();

  const handleConfirmDelete = () => {
    deleteSingle({ namespace, name }, { onSuccess: () => setShowDeleteModal(false) });
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
            onClick={() => openTab("modification", { kind: "Lease", name, namespace })}
          />
          <ResourceDeletionButton onClick={() => setShowDeleteModal(true)} />
        </DropdownMenuContent>
      </DropdownMenu>

      <LeaseDeleteConfirmationModal
        open={showDeleteModal}
        mode="single"
        name={name}
        namespace={namespace}
        isPending={isDeletePending}
        onClose={() => setShowDeleteModal(false)}
        onConfirm={handleConfirmDelete}
      />
    </>
  );
};

export const LeasesView: FC = () => {
  const [search, setSearch] = useState("");
  const [selection, setSelection] = useState<Set<string>>(new Set());
  const [showBulkDeleteModal, setShowBulkDeleteModal] = useState(false);

  const { activeContext, namespace } = useMainLayoutContext();
  const { onToggleNamespaceDetail, onToggleLease } = useDetailDrawerContext();
  const { mutate: deleteBulk, isPending: isDeleteBulkPending } = useDeleteLeases();

  const { data: raw = [], isLoading } = useGetLeases({ context: activeContext, namespace });

  const leases = raw
    .filter((l) => !search || l.Name.toLowerCase().includes(search.toLowerCase()))
    .toSorted((a, b) => a.Name.localeCompare(b.Name));

  const allLeaseKeys = useMemo(
    () => new Set(leases.map((l) => `${l.Namespace}/${l.Name}`)),
    [leases]
  );

  const handleSelectAll = () => {
    if (selection.size === allLeaseKeys.size) {
      setSelection(new Set());
    } else {
      setSelection(new Set(allLeaseKeys));
    }
  };

  const handleToggleRow = (key: string) => {
    const newSelection = new Set(selection);
    if (newSelection.has(key)) {
      newSelection.delete(key);
    } else {
      newSelection.add(key);
    }
    setSelection(newSelection);
  };

  const handleBulkDeleteClick = () => {
    if (selection.size === 0) return;
    setShowBulkDeleteModal(true);
  };

  const handleConfirmBulkDelete = () => {
    const items = Array.from(selection).map((key) => {
      const [ns, name] = key.split("/");
      return { namespace: ns, name };
    });
    deleteBulk(
      { items },
      {
        onSuccess: () => {
          setShowBulkDeleteModal(false);
          setSelection(new Set());
        },
      }
    );
  };

  return (
    <div className="flex h-full flex-col gap-3">
      <div className="flex items-center gap-3">
        <span className="text-h1">Leases</span>
        <span className="text-muted-foreground text-xs">
          {leases.length} item{leases.length === 1 ? "" : "s"}
        </span>
        <div className="ml-auto flex items-center gap-4">
          <ResourceBulkDeletionButton
            count={selection.size}
            ariaLabel="Delete selected leases"
            tooltip="Delete selected Leases"
            onClick={handleBulkDeleteClick}
          />
          <SearchInput
            placeholder="Search Leases..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            wrapperClassName="w-68"
          />
        </div>
      </div>

      <Table containerClassName="flex-1 overflow-y-auto">
        <TableHeader className="bg-background z-sticky sticky top-0">
          <TableRow>
            <TableHead className="w-8">
              <Checkbox
                checked={selection.size === allLeaseKeys.size && allLeaseKeys.size > 0}
                indeterminate={selection.size > 0 && selection.size < allLeaseKeys.size}
                onCheckedChange={() => handleSelectAll()}
              />
            </TableHead>
            <TableHead>Name</TableHead>
            {!namespace && <TableHead>Namespace</TableHead>}
            <TableHead>Holder</TableHead>
            <TableHead>Age</TableHead>
            <TableHead className="w-8" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            <TableSkeletonLoader
              rows={5}
              columns={namespace ? 3 : 4}
              includeCheckbox={true}
              columnWidths={["w-[65%]", "w-[55%]", "w-[45%]", "w-[30%]"]}
            />
          ) : leases.length === 0 ? (
            <TableRow>
              <TableCell colSpan={namespace ? 5 : 6} className="px-0 py-0">
                <EmptyState
                  icon={<ClockIcon className="size-8" />}
                  title="No Leases"
                  description="Leases are used for leader election and heartbeats"
                />
              </TableCell>
            </TableRow>
          ) : (
            leases.map((lease) => {
              const key = `${lease.Namespace}/${lease.Name}`;
              return (
                <TableRow
                  key={key}
                  onClick={() => {
                    onToggleLease(lease.Namespace, lease.Name);
                  }}
                >
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <Checkbox
                      checked={selection.has(key)}
                      onCheckedChange={() => {
                        handleToggleRow(key);
                      }}
                    />
                  </TableCell>
                  <TableCell className="font-mono text-xs">{lease.Name}</TableCell>
                  {!namespace && (
                    <TableCell>
                      <span className="text-xs">
                        <ResourceLink
                          onClick={(e) => {
                            e.stopPropagation();
                            onToggleNamespaceDetail(lease.Namespace);
                          }}
                        >
                          {lease.Namespace}
                        </ResourceLink>
                      </span>
                    </TableCell>
                  )}
                  <TableCell
                    className="text-muted-foreground max-w-xs truncate font-mono text-xs"
                    title={lease.HolderIdentity}
                  >
                    {lease.HolderIdentity || "—"}
                  </TableCell>
                  <TableCell className="text-xs">{lease.Age}</TableCell>
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <LeaseTableCtaButtons namespace={lease.Namespace} name={lease.Name} />
                  </TableCell>
                </TableRow>
              );
            })
          )}
        </TableBody>
      </Table>

      <LeaseDeleteConfirmationModal
        open={showBulkDeleteModal}
        mode="bulk"
        items={Array.from(selection).map((key) => {
          const [ns, name] = key.split("/");
          return { namespace: ns, name };
        })}
        isPending={isDeleteBulkPending}
        onClose={() => setShowBulkDeleteModal(false)}
        onConfirm={handleConfirmBulkDelete}
      />
    </div>
  );
};
