import {
  Checkbox,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
  EmptyState,
  MoreVerticalIcon,
  ResourceBulkDeletionButton,
  ResourceDeletionButton,
  ResourceLink,
  ResourceModificationButton,
  ScalingIcon,
  SearchInput,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  TableSkeletonLoader,
} from "@litelens/design-system";
import { FC, useState } from "react";
import { useGetHPAs } from "./hooks/data-access/useGetHPAs";
import { useDeleteHPA } from "./hooks/data-mutation/useDeleteHPA";
import { useDeleteHPAs } from "./hooks/data-mutation/useDeleteHPAs";
import { useMainLayoutContext } from "../../../MainLayoutContext";
import { useDetailDrawerContext } from "../../../shared/components/details/DetailDrawerContext";
import { useUnifiedTray } from "../../../shared/components/trays/unified/UnifiedTrayContext";
import { HPADeleteConfirmationModal } from "./components/HPADeleteConfirmationModal";
import { HPAStatusBadge } from "./components/HPAStatusBadge";

interface HPATableCtaButtonsProps {
  name: string;
  namespace: string;
}

const HPATableCtaButtons: FC<HPATableCtaButtonsProps> = ({ namespace, name }) => {
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const { openTab } = useUnifiedTray();

  const { mutate: deleteHPA, isPending: isDeletePending } = useDeleteHPA();

  const handleDeleteConfirm = () => {
    deleteHPA({ namespace, name }, { onSuccess: () => setShowDeleteModal(false) });
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
            onClick={() => openTab("modification", { kind: "HPA", name, namespace })}
          />
          <ResourceDeletionButton
            disabled={isDeletePending}
            onClick={() => setShowDeleteModal(true)}
          />
        </DropdownMenuContent>
      </DropdownMenu>

      <HPADeleteConfirmationModal
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

export const HPAView: FC = () => {
  const [search, setSearch] = useState("");
  const [selectedHPAIds, setSelectedHPAIds] = useState<Set<string>>(new Set());
  const [showBulkDeleteModal, setShowBulkDeleteModal] = useState(false);

  const { activeContext, namespaces } = useMainLayoutContext();
  const { onToggleNamespaceDetail, onToggleHPADetail } = useDetailDrawerContext();

  const { mutate: deleteHPAs, isPending: isBulkDeletePending } = useDeleteHPAs();

  const { data: raw = [], isLoading } = useGetHPAs({ context: activeContext, namespaces });

  const hpas = raw
    .filter((h) => !search || h.Name.toLowerCase().includes(search.toLowerCase()))
    .toSorted((a, b) => a.Name.localeCompare(b.Name));

  return (
    <div className="flex h-full flex-col gap-3">
      <div className="flex items-center gap-3">
        <span className="text-h1">Horizontal Pod Autoscalers</span>
        <span className="text-muted-foreground text-xs">
          {hpas.length} item{hpas.length !== 1 ? "s" : ""}
        </span>
        <div className="ml-auto flex items-center gap-4">
          <ResourceBulkDeletionButton
            count={selectedHPAIds.size}
            ariaLabel="Delete selected hpas"
            tooltip="Delete selected HPAs"
            onClick={() => setShowBulkDeleteModal(true)}
          />
          <SearchInput
            placeholder="Search Horizontal Pod Autoscalers..."
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
                  hpas.length > 0 &&
                  hpas.every((h) => selectedHPAIds.has(`${h.Namespace}/${h.Name}`))
                }
                indeterminate={
                  hpas.some((h) => selectedHPAIds.has(`${h.Namespace}/${h.Name}`)) &&
                  !hpas.every((h) => selectedHPAIds.has(`${h.Namespace}/${h.Name}`))
                }
                onCheckedChange={(checked) => {
                  if (checked) {
                    const newSelection = new Set(selectedHPAIds);
                    hpas.forEach((h) => newSelection.add(`${h.Namespace}/${h.Name}`));
                    setSelectedHPAIds(newSelection);
                  } else {
                    const newSelection = new Set(selectedHPAIds);
                    hpas.forEach((h) => newSelection.delete(`${h.Namespace}/${h.Name}`));
                    setSelectedHPAIds(newSelection);
                  }
                }}
              />
            </TableHead>
            <TableHead>Name</TableHead>
            {namespaces.length !== 1 && <TableHead>Namespace</TableHead>}
            <TableHead>Metrics</TableHead>
            <TableHead>Min Pods</TableHead>
            <TableHead>Max Pods</TableHead>
            <TableHead>Replicas</TableHead>
            <TableHead>Age</TableHead>
            <TableHead>Status</TableHead>
            <TableHead className="w-8" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            <TableSkeletonLoader
              rows={5}
              columns={namespaces.length !== 1 ? 7 : 6}
              includeCheckbox={true}
              columnWidths={[
                "w-[65%]",
                "w-[55%]",
                "w-[45%]",
                "w-[30%]",
                "w-[30%]",
                "w-[35%]",
                "w-[30%]",
              ]}
            />
          ) : hpas.length === 0 ? (
            <TableRow>
              <TableCell colSpan={namespaces.length !== 1 ? 10 : 9} className="px-0 py-0">
                <EmptyState
                  icon={<ScalingIcon className="size-8" />}
                  title="No HorizontalPodAutoscalers"
                  description="Create an HPA to automatically scale workloads"
                />
              </TableCell>
            </TableRow>
          ) : (
            hpas.map((h) => (
              <TableRow
                key={`${h.Namespace}/${h.Name}`}
                onClick={() => onToggleHPADetail(h.Namespace, h.Name)}
              >
                <TableCell onClick={(e) => e.stopPropagation()}>
                  <Checkbox
                    checked={selectedHPAIds.has(`${h.Namespace}/${h.Name}`)}
                    onCheckedChange={(checked) => {
                      const newSelection = new Set(selectedHPAIds);
                      if (checked) {
                        newSelection.add(`${h.Namespace}/${h.Name}`);
                      } else {
                        newSelection.delete(`${h.Namespace}/${h.Name}`);
                      }
                      setSelectedHPAIds(newSelection);
                    }}
                  />
                </TableCell>
                <TableCell className="font-mono text-xs">{h.Name}</TableCell>
                {namespaces.length !== 1 && (
                  <TableCell className="text-xs">
                    <ResourceLink
                      onClick={(e) => {
                        e.stopPropagation();
                        onToggleNamespaceDetail(h.Namespace);
                      }}
                    >
                      {h.Namespace}
                    </ResourceLink>
                  </TableCell>
                )}
                <TableCell className="font-mono text-xs">{h.Metrics}</TableCell>
                <TableCell className="text-xs">{h.MinPods}</TableCell>
                <TableCell className="text-xs">{h.MaxPods}</TableCell>
                <TableCell className="text-xs">{h.Replicas}</TableCell>
                <TableCell className="text-xs">{h.Age}</TableCell>
                <TableCell>
                  <HPAStatusBadge status={h.Status} />
                </TableCell>
                <TableCell onClick={(e) => e.stopPropagation()}>
                  <HPATableCtaButtons name={h.Name} namespace={h.Namespace} />
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>

      {selectedHPAIds.size > 0 && (
        <HPADeleteConfirmationModal
          open={showBulkDeleteModal}
          mode="bulk"
          items={Array.from(selectedHPAIds).map((key) => {
            const [ns, name] = key.split("/");
            return { namespace: ns, name };
          })}
          isPending={isBulkDeletePending}
          onClose={() => setShowBulkDeleteModal(false)}
          onConfirm={() => {
            const items = Array.from(selectedHPAIds).map((key) => {
              const [ns, name] = key.split("/");
              return { namespace: ns, name };
            });
            deleteHPAs(
              { items },
              {
                onSuccess: () => {
                  setShowBulkDeleteModal(false);
                  setSelectedHPAIds(new Set());
                },
              }
            );
          }}
        />
      )}
    </div>
  );
};
