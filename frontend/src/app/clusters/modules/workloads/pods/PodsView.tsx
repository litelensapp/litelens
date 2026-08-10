import {
  Checkbox,
  ContainerIcon,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  EmptyState,
  MoreVerticalIcon,
  ResourceBulkDeletionButton,
  ResourceCell,
  ResourceDeletionButton,
  ResourceLink,
  ResourceModificationButton,
  ScrollTextIcon,
  SearchInput,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  TableSkeletonLoader,
  TerminalIcon,
  TruncatedText,
  cn,
} from "@litelens/design-system";
import { FC, useState } from "react";
import { useMainLayoutContext } from "../../../MainLayoutContext";
import { useDetailDrawerContext } from "../../../shared/components/details/DetailDrawerContext";
import { useUnifiedTray } from "../../../shared/components/trays/unified/UnifiedTrayContext";
import { useResourceLinks } from "../../../shared/hooks/useResourceLinks";
import type { Pod } from "./api/resources";
import { PodContainerDots } from "./components/PodContainerDots";
import { PodDeleteConfirmationModal } from "./components/PodDeleteConfirmationModal";
import { PodQoSBadge } from "./components/PodQoSBadge";
import { PodStatusBadge } from "./components/PodStatusBadge";
import { useGetPods } from "./hooks/data-access/useGetPods";
import { useDeletePod } from "./hooks/data-mutation/useDeletePod";
import { useDeletePods } from "./hooks/data-mutation/useDeletePods";

interface PodTableCtaButtonsProps {
  name: string;
  namespace: string;
  onLogs: () => void;
  onExec: () => void;
}

const PodTableCtaButtons: FC<PodTableCtaButtonsProps> = ({ name, namespace, onLogs, onExec }) => {
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const { openTab } = useUnifiedTray();

  const { mutate: deletePod, isPending: isDeletePending } = useDeletePod();

  const handleDeleteConfirm = () => {
    deletePod({ namespace, name }, { onSuccess: () => setShowDeleteModal(false) });
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
          <DropdownMenuItem onClick={onLogs}>
            <ScrollTextIcon className="mr-2 size-3.5" />
            Logs
          </DropdownMenuItem>
          <DropdownMenuItem onClick={onExec}>
            <TerminalIcon className="mr-2 size-3.5" />
            Exec
          </DropdownMenuItem>
          <ResourceModificationButton
            onClick={() => openTab("modification", { kind: "Pod", name, namespace })}
          />
          <ResourceDeletionButton
            disabled={isDeletePending}
            onClick={() => setShowDeleteModal(true)}
          />
        </DropdownMenuContent>
      </DropdownMenu>

      <PodDeleteConfirmationModal
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

export const PodsView: FC = () => {
  const { activeContext, namespace } = useMainLayoutContext();
  const { onToggleNamespaceDetail, onTogglePodDetail } = useDetailDrawerContext();

  const resourceLinks = useResourceLinks();

  const [search, setSearch] = useState("");
  const [selectedPodIds, setSelectedPodIds] = useState<Set<string>>(new Set());
  const [showBulkDeleteModal, setShowBulkDeleteModal] = useState(false);

  const { mutate: deletePods, isPending: isBulkDeletePending } = useDeletePods();
  const { openTab } = useUnifiedTray();

  const openTray = (pod: Pod, mode: "logs" | "exec") => {
    openTab("pod", {
      contextName: activeContext,
      ns: pod.Namespace,
      pod: pod.Name,
      containers: pod.ContainerDetails ?? [],
      mode,
      ownerKind: pod.ControlledBy || undefined,
      ownerName: pod.ControlledByName || undefined,
    });
  };

  const { data: raw = [], isLoading } = useGetPods({ context: activeContext, namespace });

  const pods = raw
    .filter((pod) => !search || pod.Name.toLowerCase().includes(search.toLowerCase()))
    .toSorted((a, b) => a.Name.localeCompare(b.Name));

  return (
    <div className="flex h-full flex-col gap-3">
      <div className="flex items-center gap-3">
        <span className="text-h1">Pods</span>
        <span className="text-muted-foreground text-xs">
          {pods.length} item{pods.length !== 1 ? "s" : ""}
        </span>
        <div className="ml-auto flex items-center gap-4">
          <ResourceBulkDeletionButton
            count={selectedPodIds.size}
            ariaLabel="Delete selected pods"
            tooltip="Delete selected pods"
            onClick={() => setShowBulkDeleteModal(true)}
          />
          <SearchInput
            placeholder="Search Pods..."
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
                  pods.length > 0 &&
                  pods.every((p) => selectedPodIds.has(`${p.Namespace}/${p.Name}`))
                }
                indeterminate={
                  pods.some((p) => selectedPodIds.has(`${p.Namespace}/${p.Name}`)) &&
                  !pods.every((p) => selectedPodIds.has(`${p.Namespace}/${p.Name}`))
                }
                onCheckedChange={(checked) => {
                  if (checked) {
                    const newSelection = new Set(selectedPodIds);
                    pods.forEach((p) => newSelection.add(`${p.Namespace}/${p.Name}`));
                    setSelectedPodIds(newSelection);
                  } else {
                    const newSelection = new Set(selectedPodIds);
                    pods.forEach((p) => newSelection.delete(`${p.Namespace}/${p.Name}`));
                    setSelectedPodIds(newSelection);
                  }
                }}
                aria-label="Select all visible pods"
              />
            </TableHead>
            <TableHead>Name</TableHead>
            {!namespace && <TableHead>Namespace</TableHead>}
            <TableHead>Containers</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Restarts</TableHead>
            <TableHead>Controlled By</TableHead>
            <TableHead>QoS</TableHead>
            <TableHead>CPU</TableHead>
            <TableHead>Memory</TableHead>
            <TableHead>Disk</TableHead>
            <TableHead>Age</TableHead>
            <TableHead />
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            <TableSkeletonLoader
              rows={5}
              columns={!namespace ? 11 : 10}
              includeCheckbox={true}
              columnWidths={[
                "w-[65%]",
                "w-[55%]",
                "w-[35%]",
                "w-[40%]",
                "w-[30%]",
                "w-[45%]",
                "w-[35%]",
                "w-[40%]",
                "w-[40%]",
                "w-[40%]",
                "w-[30%]",
              ]}
            />
          ) : pods.length === 0 ? (
            <TableRow>
              <TableCell colSpan={!namespace ? 13 : 12} className="px-0 py-0">
                <EmptyState
                  icon={<ContainerIcon className="size-8" />}
                  title="No Pods"
                  description="Create a pod to get started"
                />
              </TableCell>
            </TableRow>
          ) : (
            pods.map((pod) => {
              const podId = `${pod.Namespace}/${pod.Name}`;
              const isSelected = selectedPodIds.has(podId);
              return (
                <TableRow
                  key={podId}
                  className={cn(isSelected && "bg-accent/30")}
                  onClick={() => onTogglePodDetail(pod.Namespace, pod.Name)}
                >
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <Checkbox
                      checked={isSelected}
                      onCheckedChange={() => {
                        const newSelection = new Set(selectedPodIds);
                        if (isSelected) newSelection.delete(podId);
                        else newSelection.add(podId);
                        setSelectedPodIds(newSelection);
                      }}
                      aria-label={`Select pod ${pod.Name}`}
                    />
                  </TableCell>
                  <TableCell>
                    <TruncatedText text={pod.Name} />
                  </TableCell>
                  {!namespace && (
                    <TableCell className="text-xs">
                      <ResourceLink
                        onClick={(e) => {
                          e.stopPropagation();
                          onToggleNamespaceDetail(pod.Namespace);
                        }}
                      >
                        {pod.Namespace}
                      </ResourceLink>
                    </TableCell>
                  )}
                  <TableCell>
                    <PodContainerDots
                      containerDetails={pod.ContainerDetails}
                      initContainerDetails={pod.InitContainerDetails}
                    />
                  </TableCell>
                  <TableCell>
                    <PodStatusBadge status={pod.Status} />
                  </TableCell>
                  <TableCell>{pod.Restarts}</TableCell>
                  <TableCell className="text-muted-foreground text-xs">
                    {pod.ControlledBy ? (
                      resourceLinks[pod.ControlledBy.toLowerCase()] ? (
                        <ResourceLink
                          onClick={(e) => {
                            e.stopPropagation();
                            resourceLinks[pod.ControlledBy.toLowerCase()](
                              pod.Namespace,
                              pod.ControlledByName
                            );
                          }}
                        >
                          {pod.ControlledBy}
                        </ResourceLink>
                      ) : (
                        pod.ControlledBy
                      )
                    ) : (
                      "—"
                    )}
                  </TableCell>
                  <TableCell>
                    <PodQoSBadge qos={pod.QoS} />
                  </TableCell>
                  <TableCell>
                    <ResourceCell label={pod.CPU} percent={pod.CPUPercent} />
                  </TableCell>
                  <TableCell>
                    <ResourceCell label={pod.Memory} percent={pod.MemPercent} />
                  </TableCell>
                  <TableCell className="text-muted-foreground text-xs">{pod.Disk}</TableCell>
                  <TableCell>{pod.Age}</TableCell>
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <PodTableCtaButtons
                      name={pod.Name}
                      namespace={pod.Namespace}
                      onLogs={() => openTray(pod, "logs")}
                      onExec={() => openTray(pod, "exec")}
                    />
                  </TableCell>
                </TableRow>
              );
            })
          )}
        </TableBody>
      </Table>

      {selectedPodIds.size > 0 && (
        <PodDeleteConfirmationModal
          open={showBulkDeleteModal}
          mode="bulk"
          items={Array.from(selectedPodIds).map((id) => {
            const [ns, name] = id.split("/");
            return { namespace: ns, name };
          })}
          isPending={isBulkDeletePending}
          onClose={() => setShowBulkDeleteModal(false)}
          onConfirm={() => {
            const items = Array.from(selectedPodIds).map((id) => {
              const [ns, name] = id.split("/");
              return { namespace: ns, name };
            });
            deletePods(
              { items },
              {
                onSuccess: () => {
                  setShowBulkDeleteModal(false);
                  setSelectedPodIds(new Set());
                },
              }
            );
          }}
        />
      )}
    </div>
  );
};
