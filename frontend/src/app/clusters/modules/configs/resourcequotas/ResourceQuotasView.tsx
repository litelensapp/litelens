import {
  Button,
  Checkbox,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
  EmptyState,
  GaugeIcon,
  MoreVerticalIcon,
  ResourceBulkDeletionButton,
  ResourceCreationButton,
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
import { useGetResourceQuotas } from "./hooks/data-access/useGetResourceQuotas";
import { useDeleteResourceQuota } from "./hooks/data-mutation/useDeleteResourceQuota";
import { useDeleteResourceQuotas } from "./hooks/data-mutation/useDeleteResourceQuotas";
import { useMainLayoutContext } from "../../../MainLayoutContext";
import { useDetailDrawerContext } from "../../../shared/components/details/DetailDrawerContext";
import { useUnifiedTray } from "../../../shared/components/trays/unified/UnifiedTrayContext";
import { ResourceQuotaCreationModal } from "./components/ResourceQuotaCreationModal";
import { ResourceQuotaDeleteConfirmationModal } from "./components/ResourceQuotaDeleteConfirmationModal";

interface ResourceQuotaTableCtaButtonsProps {
  name: string;
  namespace: string;
}

const ResourceQuotaTableCtaButtons: FC<ResourceQuotaTableCtaButtonsProps> = ({
  name,
  namespace,
}) => {
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const { openTab } = useUnifiedTray();

  const { mutate: deleteQuota, isPending: isDeletePending } = useDeleteResourceQuota();

  const handleDeleteConfirm = () => {
    deleteQuota({ namespace, name }, { onSuccess: () => setShowDeleteModal(false) });
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
            onClick={() => openTab("modification", { kind: "ResourceQuota", name, namespace })}
          />
          <ResourceDeletionButton
            disabled={isDeletePending}
            onClick={() => setShowDeleteModal(true)}
          />
        </DropdownMenuContent>
      </DropdownMenu>

      <ResourceQuotaDeleteConfirmationModal
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

export const ResourceQuotasView: FC = () => {
  const [search, setSearch] = useState("");
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [selectedResourceQuotaIds, setSelectedResourceQuotaIds] = useState<Set<string>>(new Set());
  const [showBulkDeleteModal, setShowBulkDeleteModal] = useState(false);

  const { activeContext, namespace } = useMainLayoutContext();
  const { onToggleNamespaceDetail, onToggleResourceQuotaDetail } = useDetailDrawerContext();

  const { mutate: deleteResourceQuotas, isPending: isBulkDeletePending } =
    useDeleteResourceQuotas();

  const { data: raw = [], isLoading } = useGetResourceQuotas({ context: activeContext, namespace });

  const quotas = raw
    .filter((rq) => !search || rq.Name.toLowerCase().includes(search.toLowerCase()))
    .toSorted((a, b) => a.Name.localeCompare(b.Name));

  return (
    <div className="flex h-full flex-col gap-3">
      <div className="flex items-center gap-3">
        <span className="text-h1">Resource Quotas</span>
        <span className="text-muted-foreground text-xs">
          {quotas.length} item{quotas.length !== 1 ? "s" : ""}
        </span>
        <div className="ml-auto flex items-center gap-4">
          <ResourceBulkDeletionButton
            count={selectedResourceQuotaIds.size}
            ariaLabel="Delete selected resource quotas"
            tooltip="Delete selected Resource Quotas"
            onClick={() => setShowBulkDeleteModal(true)}
          />
          <ResourceCreationButton
            ariaLabel="Create ResourceQuota"
            tooltip="Create Resource Quota"
            onClick={() => setIsCreateOpen(true)}
          />
          <SearchInput
            placeholder="Search Resource Quotas..."
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
                  quotas.length > 0 &&
                  quotas.every((rq) => selectedResourceQuotaIds.has(`${rq.Namespace}/${rq.Name}`))
                }
                indeterminate={
                  quotas.some((rq) => selectedResourceQuotaIds.has(`${rq.Namespace}/${rq.Name}`)) &&
                  !quotas.every((rq) => selectedResourceQuotaIds.has(`${rq.Namespace}/${rq.Name}`))
                }
                onCheckedChange={(checked) => {
                  if (checked) {
                    const newSelection = new Set(selectedResourceQuotaIds);
                    quotas.forEach((rq) => newSelection.add(`${rq.Namespace}/${rq.Name}`));
                    setSelectedResourceQuotaIds(newSelection);
                  } else {
                    const newSelection = new Set(selectedResourceQuotaIds);
                    quotas.forEach((rq) => newSelection.delete(`${rq.Namespace}/${rq.Name}`));
                    setSelectedResourceQuotaIds(newSelection);
                  }
                }}
                aria-label="Select all visible resource quotas"
              />
            </TableHead>
            <TableHead>Name</TableHead>
            {!namespace && <TableHead>Namespace</TableHead>}
            <TableHead>Age</TableHead>
            <TableHead className="w-8" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            <TableSkeletonLoader
              rows={5}
              columns={namespace ? 2 : 3}
              includeCheckbox={true}
              columnWidths={["w-[65%]", "w-[55%]", "w-[30%]"]}
            />
          ) : quotas.length === 0 ? (
            <TableRow>
              <TableCell colSpan={namespace ? 4 : 5} className="px-0 py-0">
                <EmptyState
                  icon={<GaugeIcon className="size-8" />}
                  title="No Resource Quotas"
                  description="Create a ResourceQuota to constrain namespace resource usage"
                  action={
                    <Button variant="default" size="default" onClick={() => setIsCreateOpen(true)}>
                      Create Resource Quota
                    </Button>
                  }
                />
              </TableCell>
            </TableRow>
          ) : (
            quotas.map((rq) => {
              const rqId = `${rq.Namespace}/${rq.Name}`;
              const isSelected = selectedResourceQuotaIds.has(rqId);
              return (
                <TableRow
                  key={rqId}
                  className={cn(isSelected && "bg-accent/30")}
                  onClick={() => onToggleResourceQuotaDetail(rq.Namespace, rq.Name)}
                >
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <Checkbox
                      checked={isSelected}
                      onCheckedChange={() => {
                        const newSelection = new Set(selectedResourceQuotaIds);
                        if (isSelected) newSelection.delete(rqId);
                        else newSelection.add(rqId);
                        setSelectedResourceQuotaIds(newSelection);
                      }}
                      aria-label={`Select resource quota ${rq.Name}`}
                    />
                  </TableCell>
                  <TableCell className="font-mono text-xs">{rq.Name}</TableCell>
                  {!namespace && (
                    <TableCell className="text-xs">
                      <ResourceLink
                        onClick={(e) => {
                          e.stopPropagation();
                          onToggleNamespaceDetail(rq.Namespace);
                        }}
                      >
                        {rq.Namespace}
                      </ResourceLink>
                    </TableCell>
                  )}
                  <TableCell className="text-xs">{rq.Age}</TableCell>
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <ResourceQuotaTableCtaButtons name={rq.Name} namespace={rq.Namespace} />
                  </TableCell>
                </TableRow>
              );
            })
          )}
        </TableBody>
      </Table>

      {selectedResourceQuotaIds.size > 0 && (
        <ResourceQuotaDeleteConfirmationModal
          open={showBulkDeleteModal}
          mode="bulk"
          items={Array.from(selectedResourceQuotaIds).map((id) => {
            const [ns, name] = id.split("/");
            return { namespace: ns, name };
          })}
          isPending={isBulkDeletePending}
          onClose={() => setShowBulkDeleteModal(false)}
          onConfirm={() => {
            const items = Array.from(selectedResourceQuotaIds).map((id) => {
              const [ns, name] = id.split("/");
              return { namespace: ns, name };
            });
            deleteResourceQuotas(
              { items },
              {
                onSuccess: () => {
                  setShowBulkDeleteModal(false);
                  setSelectedResourceQuotaIds(new Set());
                },
              }
            );
          }}
        />
      )}

      <ResourceQuotaCreationModal
        open={isCreateOpen}
        onClose={() => setIsCreateOpen(false)}
        activeNamespace={namespace ?? ""}
        activeContext={activeContext}
      />
    </div>
  );
};
