import {
  Checkbox,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
  EmptyState,
  MoreVerticalIcon,
  NetworkIcon,
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
  TruncatedText,
  cn,
} from "@litelens/design-system";
import { FC, useState } from "react";
import { useGetEndpointSlices } from "./hooks/data-access/useGetEndpointSlices";
import { useDeleteEndpointSlice } from "./hooks/data-mutation/useDeleteEndpointSlice";
import { useDeleteEndpointSlices } from "./hooks/data-mutation/useDeleteEndpointSlices";
import { useMainLayoutContext } from "../../../MainLayoutContext";
import { useDetailDrawerContext } from "../../../shared/components/details/DetailDrawerContext";
import { useUnifiedTray } from "../../../shared/components/trays/unified/UnifiedTrayContext";
import { EndpointSliceDeleteConfirmationModal } from "./components/EndpointSliceDeleteConfirmationModal";

interface EndpointSliceTableCtaButtonsProps {
  name: string;
  namespace: string;
}

const EndpointSliceTableCtaButtons: FC<EndpointSliceTableCtaButtonsProps> = ({
  name,
  namespace,
}) => {
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const { openTab } = useUnifiedTray();

  const { mutate: deleteEndpointSlice, isPending: isDeletePending } = useDeleteEndpointSlice();

  const handleDeleteConfirm = () => {
    deleteEndpointSlice({ namespace, name }, { onSuccess: () => setShowDeleteModal(false) });
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
            onClick={() => openTab("modification", { kind: "EndpointSlice", name, namespace })}
          />
          <ResourceDeletionButton
            disabled={isDeletePending}
            onClick={() => setShowDeleteModal(true)}
          />
        </DropdownMenuContent>
      </DropdownMenu>

      <EndpointSliceDeleteConfirmationModal
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

export const EndpointSlicesView: FC = () => {
  const [search, setSearch] = useState("");
  const [selectedSliceIds, setSelectedSliceIds] = useState<Set<string>>(new Set());
  const [showBulkDeleteModal, setShowBulkDeleteModal] = useState(false);

  const { activeContext, namespace } = useMainLayoutContext();
  const { onToggleNamespaceDetail, onToggleEndpointSliceDetail } = useDetailDrawerContext();

  const { mutate: deleteEndpointSlices, isPending: isBulkDeletePending } =
    useDeleteEndpointSlices();

  const { data: raw = [], isLoading } = useGetEndpointSlices({ context: activeContext, namespace });

  const slices = raw
    .filter((s) => !search || s.Name.toLowerCase().includes(search.toLowerCase()))
    .toSorted((a, b) => a.Name.localeCompare(b.Name));

  return (
    <div className="flex h-full flex-col gap-3">
      <div className="flex items-center gap-3">
        <span className="text-h1">Endpoint Slices</span>
        <span className="text-muted-foreground text-xs">
          {slices.length} item{slices.length === 1 ? "" : "s"}
        </span>
        <div className="ml-auto flex items-center gap-4">
          <ResourceBulkDeletionButton
            count={selectedSliceIds.size}
            ariaLabel="Delete selected endpoint slices"
            tooltip="Delete selected EndpointSlices"
            onClick={() => setShowBulkDeleteModal(true)}
          />
          <SearchInput
            aria-label="Search Endpoint Slices"
            placeholder="Search Endpoint Slices..."
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
                  slices.length > 0 &&
                  slices.every((slice) => selectedSliceIds.has(`${slice.Namespace}/${slice.Name}`))
                }
                indeterminate={
                  slices.some((slice) =>
                    selectedSliceIds.has(`${slice.Namespace}/${slice.Name}`)
                  ) &&
                  !slices.every((slice) => selectedSliceIds.has(`${slice.Namespace}/${slice.Name}`))
                }
                onCheckedChange={(checked) => {
                  if (checked) {
                    const newSelection = new Set(selectedSliceIds);
                    slices.forEach((slice) => newSelection.add(`${slice.Namespace}/${slice.Name}`));
                    setSelectedSliceIds(newSelection);
                  } else {
                    const newSelection = new Set(selectedSliceIds);
                    slices.forEach((slice) =>
                      newSelection.delete(`${slice.Namespace}/${slice.Name}`)
                    );
                    setSelectedSliceIds(newSelection);
                  }
                }}
                aria-label="Select all visible endpoint slices"
              />
            </TableHead>
            <TableHead>Name</TableHead>
            {!namespace && <TableHead>Namespace</TableHead>}
            <TableHead>Address Type</TableHead>
            <TableHead>Ports</TableHead>
            <TableHead>Endpoints</TableHead>
            <TableHead>Age</TableHead>
            <TableHead className="w-8" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            <TableSkeletonLoader
              rows={5}
              columns={namespace ? 4 : 5}
              includeCheckbox={true}
              columnWidths={["w-[65%]", "w-[55%]", "w-[35%]", "w-[45%]", "w-[50%]"]}
            />
          ) : (
            <>
              {slices.map((slice) => {
                const sliceId = `${slice.Namespace}/${slice.Name}`;
                const isSelected = selectedSliceIds.has(sliceId);
                const ports = (slice.Ports ?? []).map((p) => `${p.Port}/${p.Protocol}`).join(", ");
                const endpoints = (slice.Endpoints ?? [])
                  .flatMap((ep) => (ep.Addresses?.[0] ? [ep.Addresses[0]] : []))
                  .join(", ");

                return (
                  <TableRow
                    key={sliceId}
                    className={cn(isSelected && "bg-accent/30", "cursor-pointer")}
                    onClick={() => onToggleEndpointSliceDetail(slice.Namespace, slice.Name)}
                  >
                    <TableCell onClick={(e) => e.stopPropagation()}>
                      <Checkbox
                        checked={isSelected}
                        onCheckedChange={() => {
                          const newSelection = new Set(selectedSliceIds);
                          if (isSelected) newSelection.delete(sliceId);
                          else newSelection.add(sliceId);
                          setSelectedSliceIds(newSelection);
                        }}
                        aria-label={`Select endpoint slice ${slice.Name}`}
                      />
                    </TableCell>
                    <TableCell className="max-w-60">
                      <TruncatedText text={slice.Name} />
                    </TableCell>
                    {!namespace && (
                      <TableCell className="text-xs">
                        <ResourceLink
                          onClick={(e) => {
                            e.stopPropagation();
                            onToggleNamespaceDetail(slice.Namespace);
                          }}
                        >
                          {slice.Namespace}
                        </ResourceLink>
                      </TableCell>
                    )}
                    <TableCell className="text-xs">{slice.AddressType}</TableCell>
                    <TableCell className="font-mono text-xs">{ports || "—"}</TableCell>
                    <TableCell className="font-mono text-xs">{endpoints || "—"}</TableCell>
                    <TableCell className="text-xs">{slice.Age}</TableCell>
                    <TableCell onClick={(e) => e.stopPropagation()}>
                      <EndpointSliceTableCtaButtons name={slice.Name} namespace={slice.Namespace} />
                    </TableCell>
                  </TableRow>
                );
              })}
              {slices.length === 0 && (
                <TableRow>
                  <TableCell colSpan={namespace ? 7 : 8} className="px-0 py-0">
                    <EmptyState
                      icon={<NetworkIcon className="size-8" />}
                      title="No EndpointSlices"
                      description="EndpointSlices are created automatically by Services"
                    />
                  </TableCell>
                </TableRow>
              )}
            </>
          )}
        </TableBody>
      </Table>

      {selectedSliceIds.size > 0 && (
        <EndpointSliceDeleteConfirmationModal
          open={showBulkDeleteModal}
          mode="bulk"
          items={Array.from(selectedSliceIds).map((id) => {
            const [ns, name] = id.split("/");
            return { namespace: ns, name };
          })}
          isPending={isBulkDeletePending}
          onClose={() => setShowBulkDeleteModal(false)}
          onConfirm={() => {
            const items = Array.from(selectedSliceIds).map((id) => {
              const [ns, name] = id.split("/");
              return { namespace: ns, name };
            });
            deleteEndpointSlices(
              { items },
              {
                onSuccess: () => {
                  setShowBulkDeleteModal(false);
                  setSelectedSliceIds(new Set());
                },
              }
            );
          }}
        />
      )}
    </div>
  );
};
