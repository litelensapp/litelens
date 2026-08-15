import {
  Checkbox,
  DatabaseIcon,
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
  cn,
} from "@litelens/design-system";
import { FC, useMemo, useState } from "react";
import { useGetStatefulSets } from "./hooks/data-access/useGetStatefulSets";
import { useDeleteStatefulSet } from "./hooks/data-mutation/useDeleteStatefulSet";
import { useDeleteStatefulSets } from "./hooks/data-mutation/useDeleteStatefulSets";
import { useMainLayoutContext } from "../../../MainLayoutContext";
import { useDetailDrawerContext } from "../../../shared/components/details/DetailDrawerContext";
import { useUnifiedTray } from "../../../shared/components/trays/unified/UnifiedTrayContext";
import { StatefulSetDeleteConfirmationModal } from "./components/StatefulSetDeleteConfirmationModal";

interface StatefulSetTableCtaButtonsProps {
  namespace: string;
  name: string;
}

const StatefulSetTableCtaButtons: FC<StatefulSetTableCtaButtonsProps> = ({ namespace, name }) => {
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const { openTab } = useUnifiedTray();

  const { mutate: deleteStatefulSet, isPending: isDeletePending } = useDeleteStatefulSet();

  const handleDeleteConfirm = () => {
    deleteStatefulSet({ namespace, name }, { onSuccess: () => setShowDeleteModal(false) });
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
            onClick={() => openTab("modification", { kind: "StatefulSet", name, namespace })}
          />
          <ResourceDeletionButton
            disabled={isDeletePending}
            onClick={() => setShowDeleteModal(true)}
          />
        </DropdownMenuContent>
      </DropdownMenu>

      <StatefulSetDeleteConfirmationModal
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

export const StatefulSetsView: FC = () => {
  const [search, setSearch] = useState("");
  const [selectedStatefulSetIds, setSelectedStatefulSetIds] = useState<Set<string>>(new Set());
  const [showBulkDeleteModal, setShowBulkDeleteModal] = useState(false);

  const { activeContext, namespaces } = useMainLayoutContext();
  const { onToggleNamespaceDetail, onToggleStatefulSetDetail } = useDetailDrawerContext();

  const { mutate: deleteStatefulSets, isPending: isBulkDeletePending } = useDeleteStatefulSets();

  const { data: raw = [], isLoading } = useGetStatefulSets({ context: activeContext, namespaces });

  const statefulsets = useMemo(
    () =>
      raw
        .filter((ss) => !search || ss.Name.toLowerCase().includes(search.toLowerCase()))
        .toSorted((a, b) => a.Name.localeCompare(b.Name)),
    [raw, search]
  );

  return (
    <div className="flex h-full flex-col gap-3">
      <div className="flex items-center gap-3">
        <span className="text-h1">Stateful Sets</span>
        <span className="text-muted-foreground text-xs">
          {statefulsets.length} item{statefulsets.length !== 1 ? "s" : ""}
        </span>
        <div className="ml-auto flex items-center gap-4">
          <ResourceBulkDeletionButton
            count={selectedStatefulSetIds.size}
            ariaLabel="Delete selected stateful sets"
            tooltip="Delete selected StatefulSets"
            onClick={() => setShowBulkDeleteModal(true)}
          />
          <SearchInput
            placeholder="Search Stateful Sets..."
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
                  statefulsets.length > 0 &&
                  statefulsets.every((ss) =>
                    selectedStatefulSetIds.has(`${ss.Namespace}/${ss.Name}`)
                  )
                }
                indeterminate={
                  statefulsets.some((ss) =>
                    selectedStatefulSetIds.has(`${ss.Namespace}/${ss.Name}`)
                  ) &&
                  !statefulsets.every((ss) =>
                    selectedStatefulSetIds.has(`${ss.Namespace}/${ss.Name}`)
                  )
                }
                onCheckedChange={(checked) => {
                  if (checked) {
                    const newSelection = new Set(selectedStatefulSetIds);
                    statefulsets.forEach((ss) => newSelection.add(`${ss.Namespace}/${ss.Name}`));
                    setSelectedStatefulSetIds(newSelection);
                  } else {
                    const newSelection = new Set(selectedStatefulSetIds);
                    statefulsets.forEach((ss) => newSelection.delete(`${ss.Namespace}/${ss.Name}`));
                    setSelectedStatefulSetIds(newSelection);
                  }
                }}
                aria-label="Select all visible stateful sets"
              />
            </TableHead>
            <TableHead>Name</TableHead>
            {namespaces.length !== 1 && <TableHead>Namespace</TableHead>}
            <TableHead>Pods</TableHead>
            <TableHead>Replicas</TableHead>
            <TableHead>Age</TableHead>
            <TableHead className="w-8" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            <TableSkeletonLoader
              rows={5}
              columns={namespaces.length !== 1 ? 5 : 4}
              includeCheckbox={true}
              columnWidths={["w-[65%]", "w-[55%]", "w-[35%]", "w-[40%]", "w-[30%]"]}
            />
          ) : statefulsets.length === 0 ? (
            <TableRow>
              <TableCell colSpan={namespaces.length !== 1 ? 7 : 6} className="px-0 py-0">
                <EmptyState
                  icon={<DatabaseIcon className="size-8" />}
                  title="No StatefulSets"
                  description="Create a StatefulSet for persistent workloads"
                />
              </TableCell>
            </TableRow>
          ) : (
            statefulsets.map((ss) => {
              const ssId = `${ss.Namespace}/${ss.Name}`;
              const isSelected = selectedStatefulSetIds.has(ssId);
              return (
                <TableRow
                  key={ssId}
                  className={cn(isSelected && "bg-accent/30", "cursor-pointer")}
                  onClick={() => onToggleStatefulSetDetail(ss.Namespace, ss.Name)}
                >
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <Checkbox
                      checked={isSelected}
                      onCheckedChange={() => {
                        const newSelection = new Set(selectedStatefulSetIds);
                        if (isSelected) newSelection.delete(ssId);
                        else newSelection.add(ssId);
                        setSelectedStatefulSetIds(newSelection);
                      }}
                      aria-label={`Select stateful set ${ss.Name}`}
                    />
                  </TableCell>
                  <TableCell className="font-mono text-xs">{ss.Name}</TableCell>
                  {namespaces.length !== 1 && (
                    <TableCell className="text-xs">
                      <ResourceLink
                        onClick={(e) => {
                          e.stopPropagation();
                          onToggleNamespaceDetail(ss.Namespace);
                        }}
                      >
                        {ss.Namespace}
                      </ResourceLink>
                    </TableCell>
                  )}
                  <TableCell className="text-xs">{ss.Pods}</TableCell>
                  <TableCell className="text-xs">{ss.Replicas}</TableCell>
                  <TableCell className="text-xs">{ss.Age}</TableCell>
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <StatefulSetTableCtaButtons namespace={ss.Namespace} name={ss.Name} />
                  </TableCell>
                </TableRow>
              );
            })
          )}
        </TableBody>
      </Table>

      {selectedStatefulSetIds.size > 0 && (
        <StatefulSetDeleteConfirmationModal
          open={showBulkDeleteModal}
          mode="bulk"
          items={Array.from(selectedStatefulSetIds).map((id) => {
            const [ns, name] = id.split("/");
            return { namespace: ns, name };
          })}
          isPending={isBulkDeletePending}
          onClose={() => setShowBulkDeleteModal(false)}
          onConfirm={() => {
            const items = Array.from(selectedStatefulSetIds).map((id) => {
              const [ns, name] = id.split("/");
              return { namespace: ns, name };
            });
            deleteStatefulSets(
              { items },
              {
                onSuccess: () => {
                  setShowBulkDeleteModal(false);
                  setSelectedStatefulSetIds(new Set());
                },
              }
            );
          }}
        />
      )}
    </div>
  );
};
