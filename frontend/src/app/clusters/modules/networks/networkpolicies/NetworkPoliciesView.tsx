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
  SearchInput,
  ShieldCheckIcon,
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
import { useGetNetworkPolicies } from "./hooks/data-access/useGetNetworkPolicies";
import { useDeleteNetworkPolicies } from "./hooks/data-mutation/useDeleteNetworkPolicies";
import { useDeleteNetworkPolicy } from "./hooks/data-mutation/useDeleteNetworkPolicy";
import { useMainLayoutContext } from "../../../MainLayoutContext";
import { useDetailDrawerContext } from "../../../shared/components/details/DetailDrawerContext";
import { useUnifiedTray } from "../../../shared/components/trays/unified/UnifiedTrayContext";
import { NetworkPolicyDeleteConfirmationModal } from "./components/NetworkPolicyDeleteConfirmationModal";

interface NetworkPolicyTableCtaButtonsProps {
  name: string;
  namespace: string;
}

const NetworkPolicyTableCtaButtons: FC<NetworkPolicyTableCtaButtonsProps> = ({
  name,
  namespace,
}) => {
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const { openTab } = useUnifiedTray();

  const { mutate: deleteNetworkPolicy, isPending: isDeletePending } = useDeleteNetworkPolicy();

  const handleDeleteConfirm = () => {
    deleteNetworkPolicy({ namespace, name }, { onSuccess: () => setShowDeleteModal(false) });
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
            onClick={() => openTab("modification", { kind: "NetworkPolicy", name, namespace })}
          />
          <ResourceDeletionButton
            disabled={isDeletePending}
            onClick={() => setShowDeleteModal(true)}
          />
        </DropdownMenuContent>
      </DropdownMenu>

      <NetworkPolicyDeleteConfirmationModal
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

export const NetworkPoliciesView: FC = () => {
  const [search, setSearch] = useState("");
  const [selectedPolicyIds, setSelectedPolicyIds] = useState<Set<string>>(new Set());
  const [showBulkDeleteModal, setShowBulkDeleteModal] = useState(false);

  const { activeContext, namespaces } = useMainLayoutContext();
  const { onToggleNamespaceDetail, onToggleNetworkPolicyDetail } = useDetailDrawerContext();

  const { mutate: deleteNetworkPolicies, isPending: isBulkDeletePending } =
    useDeleteNetworkPolicies();

  const { data: raw = [], isLoading } = useGetNetworkPolicies({
    context: activeContext,
    namespaces,
  });

  const policies = raw
    .filter((n) => !search || n.Name.toLowerCase().includes(search.toLowerCase()))
    .toSorted((a, b) => a.Name.localeCompare(b.Name));

  return (
    <div className="flex h-full flex-col gap-3">
      <div className="flex items-center gap-3">
        <span className="text-h1">Network Policies</span>
        <span className="text-xs text-muted-foreground">
          {policies.length} item{policies.length !== 1 ? "s" : ""}
        </span>
        <div className="ml-auto flex items-center gap-4">
          <ResourceBulkDeletionButton
            count={selectedPolicyIds.size}
            ariaLabel="Delete selected network policies"
            tooltip="Delete selected NetworkPolicies"
            onClick={() => setShowBulkDeleteModal(true)}
          />
          <SearchInput
            placeholder="Search Network Policies..."
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
                  policies.length > 0 &&
                  policies.every((p) => selectedPolicyIds.has(`${p.Namespace}/${p.Name}`))
                }
                indeterminate={
                  policies.some((p) => selectedPolicyIds.has(`${p.Namespace}/${p.Name}`)) &&
                  !policies.every((p) => selectedPolicyIds.has(`${p.Namespace}/${p.Name}`))
                }
                onCheckedChange={(checked) => {
                  if (checked) {
                    const newSelection = new Set(selectedPolicyIds);
                    policies.forEach((p) => newSelection.add(`${p.Namespace}/${p.Name}`));
                    setSelectedPolicyIds(newSelection);
                  } else {
                    const newSelection = new Set(selectedPolicyIds);
                    policies.forEach((p) => newSelection.delete(`${p.Namespace}/${p.Name}`));
                    setSelectedPolicyIds(newSelection);
                  }
                }}
                aria-label="Select all visible network policies"
              />
            </TableHead>
            <TableHead>Name</TableHead>
            {namespaces.length !== 1 && <TableHead>Namespace</TableHead>}
            <TableHead>Policy Types</TableHead>
            <TableHead>Age</TableHead>
            <TableHead className="w-8" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            <TableSkeletonLoader
              rows={5}
              columns={namespaces.length !== 1 ? 3 : 2}
              includeCheckbox={true}
              columnWidths={["w-[65%]", "w-[55%]", "w-[40%]"]}
            />
          ) : policies.length === 0 ? (
            <TableRow>
              <TableCell colSpan={namespaces.length !== 1 ? 6 : 5} className="px-0 py-0">
                <EmptyState
                  icon={<ShieldCheckIcon className="size-8" />}
                  title="No NetworkPolicies"
                  description="Create a NetworkPolicy to control pod traffic"
                />
              </TableCell>
            </TableRow>
          ) : (
            policies.map((n) => {
              const npId = `${n.Namespace}/${n.Name}`;
              const isSelected = selectedPolicyIds.has(npId);
              return (
                <TableRow
                  key={npId}
                  className={cn(isSelected && "bg-accent/30", "cursor-pointer")}
                  onClick={() => onToggleNetworkPolicyDetail(n.Namespace, n.Name)}
                >
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <Checkbox
                      checked={isSelected}
                      onCheckedChange={() => {
                        const newSelection = new Set(selectedPolicyIds);
                        if (isSelected) newSelection.delete(npId);
                        else newSelection.add(npId);
                        setSelectedPolicyIds(newSelection);
                      }}
                      aria-label={`Select network policy ${n.Name}`}
                    />
                  </TableCell>
                  <TableCell className="font-mono text-xs">{n.Name}</TableCell>
                  {namespaces.length !== 1 && (
                    <TableCell className="text-xs">
                      <ResourceLink
                        onClick={(e) => {
                          e.stopPropagation();
                          onToggleNamespaceDetail(n.Namespace);
                        }}
                      >
                        {n.Namespace}
                      </ResourceLink>
                    </TableCell>
                  )}
                  <TableCell className="text-xs">{n.PolicyTypes}</TableCell>
                  <TableCell className="text-xs">{n.Age}</TableCell>
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <NetworkPolicyTableCtaButtons name={n.Name} namespace={n.Namespace} />
                  </TableCell>
                </TableRow>
              );
            })
          )}
        </TableBody>
      </Table>

      {selectedPolicyIds.size > 0 && (
        <NetworkPolicyDeleteConfirmationModal
          open={showBulkDeleteModal}
          mode="bulk"
          items={Array.from(selectedPolicyIds).map((id) => {
            const [ns, name] = id.split("/");
            return { namespace: ns, name };
          })}
          isPending={isBulkDeletePending}
          onClose={() => setShowBulkDeleteModal(false)}
          onConfirm={() => {
            const items = Array.from(selectedPolicyIds).map((id) => {
              const [ns, name] = id.split("/");
              return { namespace: ns, name };
            });
            deleteNetworkPolicies(
              { items },
              {
                onSuccess: () => {
                  setShowBulkDeleteModal(false);
                  setSelectedPolicyIds(new Set());
                },
              }
            );
          }}
        />
      )}
    </div>
  );
};
