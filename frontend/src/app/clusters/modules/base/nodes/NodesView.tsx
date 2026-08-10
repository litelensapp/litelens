import {
  Checkbox,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
  EmptyState,
  MoreVerticalIcon,
  ResourceBulkDeletionButton,
  ResourceCell,
  ResourceDeletionButton,
  ResourceModificationButton,
  SearchInput,
  ServerIcon,
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
import { useGetNodes } from "./hooks/data-access/useGetNodes";
import { useCordonNode } from "./hooks/data-mutation/useCordonNode";
import { useDeleteNode } from "./hooks/data-mutation/useDeleteNode";
import { useDeleteNodes } from "./hooks/data-mutation/useDeleteNodes";
import { useDrainNode } from "./hooks/data-mutation/useDrainNode";
import { useUncordonNode } from "./hooks/data-mutation/useUncordonNode";
import { useMainLayoutContext } from "../../../MainLayoutContext";
import { useDetailDrawerContext } from "../../../shared/components/details/DetailDrawerContext";
import { useUnifiedTray } from "../../../shared/components/trays/unified/UnifiedTrayContext";
import { NodeConditionBadge } from "./components/NodeConditionBadge";
import { NodeCordonButton } from "./components/NodeCordonButton";
import { NodeCordonConfirmationModal } from "./components/NodeCordonConfirmationModal";
import { NodeDeleteConfirmationModal } from "./components/NodeDeleteConfirmationModal";
import { NodeDrainButton } from "./components/NodeDrainButton";
import { NodeDrainConfirmationModal } from "./components/NodeDrainConfirmationModal";
import { NodeSchedulableBadge } from "./components/NodeSchedulableBadge";
import { NodeUncordonButton } from "./components/NodeUncordonButton";
import { NodeUncordonConfirmationModal } from "./components/NodeUncordonConfirmationModal";

const NodeTableCtaButtons: FC<{ name: string; unschedulable: boolean }> = ({
  name,
  unschedulable,
}) => {
  const { openTab } = useUnifiedTray();

  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [showCordonModal, setShowCordonModal] = useState(false);
  const [showUncordonModal, setShowUncordonModal] = useState(false);
  const [showDrainModal, setShowDrainModal] = useState(false);

  const { mutate: deleteNode, isPending: isDeletePending } = useDeleteNode();
  const { mutate: cordonNode, isPending: isCordonPending } = useCordonNode();
  const { mutate: uncordonNode, isPending: isUncordonPending } = useUncordonNode();
  const { mutate: drainNode, isPending: isDrainPending } = useDrainNode();

  const handleDeleteConfirm = () => {
    deleteNode({ name }, { onSuccess: () => setShowDeleteModal(false) });
  };

  const handleCordonConfirm = () => {
    cordonNode({ name }, { onSuccess: () => setShowCordonModal(false) });
  };

  const handleUncordonConfirm = () => {
    uncordonNode({ name }, { onSuccess: () => setShowUncordonModal(false) });
  };

  const handleDrainConfirm = () => {
    drainNode({ name }, { onSuccess: () => setShowDrainModal(false) });
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
          {!unschedulable ? (
            <NodeCordonButton disabled={isCordonPending} onClick={() => setShowCordonModal(true)} />
          ) : (
            <NodeUncordonButton
              disabled={isUncordonPending}
              onClick={() => setShowUncordonModal(true)}
            />
          )}
          <NodeDrainButton disabled={isDrainPending} onClick={() => setShowDrainModal(true)} />
          <ResourceModificationButton
            onClick={() => openTab("modification", { kind: "Node", name })}
          />
          <ResourceDeletionButton
            disabled={isDeletePending}
            onClick={() => setShowDeleteModal(true)}
          />
        </DropdownMenuContent>
      </DropdownMenu>

      <NodeDeleteConfirmationModal
        open={showDeleteModal}
        mode="single"
        name={name}
        isPending={isDeletePending}
        onClose={() => setShowDeleteModal(false)}
        onConfirm={handleDeleteConfirm}
      />

      <NodeCordonConfirmationModal
        open={showCordonModal}
        name={name}
        isPending={isCordonPending}
        onClose={() => setShowCordonModal(false)}
        onConfirm={handleCordonConfirm}
      />

      <NodeUncordonConfirmationModal
        open={showUncordonModal}
        name={name}
        isPending={isUncordonPending}
        onClose={() => setShowUncordonModal(false)}
        onConfirm={handleUncordonConfirm}
      />

      <NodeDrainConfirmationModal
        open={showDrainModal}
        name={name}
        isPending={isDrainPending}
        onClose={() => setShowDrainModal(false)}
        onConfirm={handleDrainConfirm}
      />
    </>
  );
};

export const NodesView: FC = () => {
  const { activeContext } = useMainLayoutContext();
  const { onToggleNodeDetail } = useDetailDrawerContext();

  const [search, setSearch] = useState("");
  const [selectedNodeNames, setSelectedNodeNames] = useState<Set<string>>(new Set());
  const [showBulkDeleteModal, setShowBulkDeleteModal] = useState(false);

  const { data: raw = [], isLoading } = useGetNodes(activeContext);

  const nodes = raw
    .filter((node) => !search || node.Name.toLowerCase().includes(search.toLowerCase()))
    .toSorted((a, b) => a.Name.localeCompare(b.Name));

  const { mutate: deleteNodes, isPending: isBulkDeletePending } = useDeleteNodes();

  const handleBulkDeleteConfirm = () => {
    const names = Array.from(selectedNodeNames);
    deleteNodes(
      { names },
      {
        onSuccess: () => {
          setShowBulkDeleteModal(false);
          setSelectedNodeNames(new Set());
        },
      }
    );
  };

  return (
    <div className="flex h-full flex-col gap-3">
      <div className="flex items-center gap-3">
        <span className="text-h1">Nodes</span>
        <span className="text-muted-foreground text-xs">
          {nodes.length} item{nodes.length === 1 ? "" : "s"}
        </span>
        <div className="ml-auto flex items-center gap-4">
          <ResourceBulkDeletionButton
            count={selectedNodeNames.size}
            ariaLabel="Delete selected nodes"
            tooltip="Delete selected Nodes"
            onClick={() => setShowBulkDeleteModal(true)}
          />
          <SearchInput
            placeholder="Search Nodes..."
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
                  nodes.length > 0 && nodes.every((node) => selectedNodeNames.has(node.Name))
                }
                indeterminate={
                  nodes.some((node) => selectedNodeNames.has(node.Name)) &&
                  !nodes.every((node) => selectedNodeNames.has(node.Name))
                }
                onCheckedChange={(checked) => {
                  if (checked) {
                    const newSelection = new Set(selectedNodeNames);
                    nodes.forEach((node) => newSelection.add(node.Name));
                    setSelectedNodeNames(newSelection);
                  } else {
                    const newSelection = new Set(selectedNodeNames);
                    nodes.forEach((node) => newSelection.delete(node.Name));
                    setSelectedNodeNames(newSelection);
                  }
                }}
                aria-label="Select all visible nodes"
              />
            </TableHead>
            <TableHead>Name</TableHead>
            <TableHead>Version</TableHead>
            <TableHead>Conditions</TableHead>
            <TableHead>Schedulable</TableHead>
            <TableHead>Roles</TableHead>
            <TableHead>CPU</TableHead>
            <TableHead>Memory</TableHead>
            <TableHead>Disk</TableHead>
            <TableHead>Taints</TableHead>
            <TableHead>Age</TableHead>
            <TableHead className="w-8" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            <TableSkeletonLoader
              rows={5}
              columns={10}
              includeCheckbox={true}
              columnWidths={[
                "w-[65%]",
                "w-[35%]",
                "w-[40%]",
                "w-[40%]",
                "w-[30%]",
                "w-[40%]",
                "w-[40%]",
                "w-[35%]",
                "w-[35%]",
                "w-[30%]",
              ]}
            />
          ) : nodes.length === 0 ? (
            <TableRow>
              <TableCell colSpan={11} className="px-0 py-0">
                <EmptyState
                  icon={<ServerIcon className="size-8" />}
                  title="No Nodes"
                  description="Nodes are added to the cluster by your infrastructure"
                />
              </TableCell>
            </TableRow>
          ) : (
            nodes.map((node) => {
              const readyCondition = node.Conditions.find((c) => c.Type === "Ready");
              return (
                <TableRow
                  key={node.Name}
                  className={cn(
                    "cursor-pointer",
                    selectedNodeNames.has(node.Name) && "bg-accent/30"
                  )}
                  onClick={() => onToggleNodeDetail(node.Name)}
                >
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <Checkbox
                      checked={selectedNodeNames.has(node.Name)}
                      onCheckedChange={(checked) => {
                        const newSelection = new Set(selectedNodeNames);
                        if (checked) {
                          newSelection.add(node.Name);
                        } else {
                          newSelection.delete(node.Name);
                        }
                        setSelectedNodeNames(newSelection);
                      }}
                      aria-label={`Select node ${node.Name}`}
                    />
                  </TableCell>
                  <TableCell className="font-mono text-xs">{node.Name}</TableCell>
                  <TableCell className="font-mono text-xs">{node.Version}</TableCell>
                  <TableCell>
                    {readyCondition && <NodeConditionBadge condition={readyCondition} />}
                  </TableCell>
                  <TableCell>
                    <NodeSchedulableBadge schedulable={!node.Unschedulable} />
                  </TableCell>
                  <TableCell>{node.Roles}</TableCell>
                  <TableCell>
                    <ResourceCell label={node.CPU} percent={node.CPUPercent} />
                  </TableCell>
                  <TableCell>
                    <ResourceCell label={node.Memory} percent={node.MemPercent} />
                  </TableCell>
                  <TableCell>
                    <ResourceCell label={node.Disk} percent={node.DiskPercent} />
                  </TableCell>
                  <TableCell>{node.Taints}</TableCell>
                  <TableCell>{node.Age}</TableCell>
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <NodeTableCtaButtons name={node.Name} unschedulable={node.Unschedulable} />
                  </TableCell>
                </TableRow>
              );
            })
          )}
        </TableBody>
      </Table>

      <NodeDeleteConfirmationModal
        open={showBulkDeleteModal}
        mode="bulk"
        items={Array.from(selectedNodeNames)}
        isPending={isBulkDeletePending}
        onClose={() => setShowBulkDeleteModal(false)}
        onConfirm={handleBulkDeleteConfirm}
      />
    </div>
  );
};
