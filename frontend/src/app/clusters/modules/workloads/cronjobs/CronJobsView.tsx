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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  TableSkeletonLoader,
  TimerIcon,
} from "@litelens/design-system";
import { FC, useMemo, useState } from "react";
import { useGetCronJobs } from "./hooks/data-access/useGetCronJobs";
import { useDeleteCronJob } from "./hooks/data-mutation/useDeleteCronJob";
import { useDeleteCronJobs } from "./hooks/data-mutation/useDeleteCronJobs";
import { useMainLayoutContext } from "../../../MainLayoutContext";
import { useDetailDrawerContext } from "../../../shared/components/details/DetailDrawerContext";
import { useUnifiedTray } from "../../../shared/components/trays/unified/UnifiedTrayContext";
import { CronJobDeleteConfirmationModal } from "./components/CronJobDeleteConfirmationModal";
import { CronJobResumedBadge } from "./components/CronJobResumedBadge";

interface CronJobTableCtaButtonsProps {
  name: string;
  namespace: string;
}

const CronJobTableCtaButtons: FC<CronJobTableCtaButtonsProps> = ({ namespace, name }) => {
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const { openTab } = useUnifiedTray();

  const { mutate: deleteCronJob, isPending: isDeletePending } = useDeleteCronJob();

  const handleDeleteConfirm = () => {
    deleteCronJob({ namespace, name }, { onSuccess: () => setShowDeleteModal(false) });
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
            onClick={() => openTab("modification", { kind: "CronJob", name, namespace })}
          />
          <ResourceDeletionButton
            disabled={isDeletePending}
            onClick={() => setShowDeleteModal(true)}
          />
        </DropdownMenuContent>
      </DropdownMenu>

      <CronJobDeleteConfirmationModal
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

export const CronJobsView: FC = () => {
  const [search, setSearch] = useState("");
  const [selectedCronJobIds, setSelectedCronJobIds] = useState<Set<string>>(new Set());
  const [showBulkDeleteModal, setShowBulkDeleteModal] = useState(false);

  const { activeContext, namespaces } = useMainLayoutContext();
  const { onToggleNamespaceDetail, onToggleCronJobDetail } = useDetailDrawerContext();

  const { mutate: deleteCronJobs, isPending: isBulkDeletePending } = useDeleteCronJobs();

  const { data: raw = [], isLoading } = useGetCronJobs({ context: activeContext, namespaces });

  const cronjobs = useMemo(
    () =>
      raw
        .filter((cj) => !search || cj.Name.toLowerCase().includes(search.toLowerCase()))
        .toSorted((a, b) => a.Name.localeCompare(b.Name)),
    [raw, search]
  );

  return (
    <div className="flex h-full flex-col gap-3">
      <div className="flex items-center gap-3">
        <span className="text-h1">Cron Jobs</span>
        <span className="text-xs text-muted-foreground">
          {cronjobs.length} item{cronjobs.length === 1 ? "" : "s"}
        </span>
        <div className="ml-auto flex items-center gap-4">
          <ResourceBulkDeletionButton
            count={selectedCronJobIds.size}
            ariaLabel="Delete selected cron jobs"
            tooltip="Delete selected CronJobs"
            onClick={() => setShowBulkDeleteModal(true)}
          />
          <SearchInput
            placeholder="Search Cron Jobs..."
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
                  cronjobs.length > 0 &&
                  cronjobs.every((cj) => selectedCronJobIds.has(`${cj.Namespace}/${cj.Name}`))
                }
                indeterminate={
                  cronjobs.some((cj) => selectedCronJobIds.has(`${cj.Namespace}/${cj.Name}`)) &&
                  !cronjobs.every((cj) => selectedCronJobIds.has(`${cj.Namespace}/${cj.Name}`))
                }
                onCheckedChange={(checked) => {
                  if (checked) {
                    const newSelection = new Set(selectedCronJobIds);
                    cronjobs.forEach((cj) => newSelection.add(`${cj.Namespace}/${cj.Name}`));
                    setSelectedCronJobIds(newSelection);
                  } else {
                    const newSelection = new Set(selectedCronJobIds);
                    cronjobs.forEach((cj) => newSelection.delete(`${cj.Namespace}/${cj.Name}`));
                    setSelectedCronJobIds(newSelection);
                  }
                }}
              />
            </TableHead>
            <TableHead>Name</TableHead>
            {namespaces.length !== 1 && <TableHead>Namespace</TableHead>}
            <TableHead>Schedule</TableHead>
            <TableHead>Timezone</TableHead>
            <TableHead>Resumed</TableHead>
            <TableHead>Active</TableHead>
            <TableHead>Last schedule</TableHead>
            <TableHead>Age</TableHead>
            <TableHead className="w-8" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            <TableSkeletonLoader
              rows={5}
              columns={namespaces.length !== 1 ? 8 : 7}
              includeCheckbox={true}
              columnWidths={[
                "w-[65%]",
                "w-[55%]",
                "w-[45%]",
                "w-[35%]",
                "w-[30%]",
                "w-[30%]",
                "w-[40%]",
                "w-[30%]",
              ]}
            />
          ) : cronjobs.length === 0 ? (
            <TableRow>
              <TableCell colSpan={namespaces.length !== 1 ? 10 : 9} className="px-0 py-0">
                <EmptyState
                  icon={<TimerIcon className="size-8" />}
                  title="No CronJobs"
                  description="Create a CronJob to run scheduled tasks"
                />
              </TableCell>
            </TableRow>
          ) : (
            cronjobs.map((cj) => (
              <TableRow
                key={`${cj.Namespace}/${cj.Name}`}
                onClick={() => onToggleCronJobDetail(cj.Namespace, cj.Name)}
              >
                <TableCell onClick={(e) => e.stopPropagation()}>
                  <Checkbox
                    checked={selectedCronJobIds.has(`${cj.Namespace}/${cj.Name}`)}
                    onCheckedChange={(checked) => {
                      const newSelection = new Set(selectedCronJobIds);
                      if (checked) {
                        newSelection.add(`${cj.Namespace}/${cj.Name}`);
                      } else {
                        newSelection.delete(`${cj.Namespace}/${cj.Name}`);
                      }
                      setSelectedCronJobIds(newSelection);
                    }}
                  />
                </TableCell>
                <TableCell className="font-mono text-xs">{cj.Name}</TableCell>
                {namespaces.length !== 1 && (
                  <TableCell className="text-xs">
                    <ResourceLink
                      onClick={(e) => {
                        e.stopPropagation();
                        onToggleNamespaceDetail(cj.Namespace);
                      }}
                    >
                      {cj.Namespace}
                    </ResourceLink>
                  </TableCell>
                )}
                <TableCell className="font-mono text-xs">{cj.Schedule}</TableCell>
                <TableCell className="text-xs text-muted-foreground">
                  {cj.Timezone || "—"}
                </TableCell>
                <TableCell className="text-xs">
                  <CronJobResumedBadge resumed={!cj.Suspend} />
                </TableCell>
                <TableCell className="text-xs">{cj.Active}</TableCell>
                <TableCell className="text-xs text-muted-foreground">
                  {cj.LastSchedule || "—"}
                </TableCell>
                <TableCell className="text-xs">{cj.Age}</TableCell>
                <TableCell onClick={(e) => e.stopPropagation()}>
                  <CronJobTableCtaButtons name={cj.Name} namespace={cj.Namespace} />
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>

      {selectedCronJobIds.size > 0 && (
        <CronJobDeleteConfirmationModal
          open={showBulkDeleteModal}
          mode="bulk"
          items={Array.from(selectedCronJobIds).map((key) => {
            const [ns, name] = key.split("/");
            return { namespace: ns, name };
          })}
          isPending={isBulkDeletePending}
          onClose={() => setShowBulkDeleteModal(false)}
          onConfirm={() => {
            const items = Array.from(selectedCronJobIds).map((key) => {
              const [ns, name] = key.split("/");
              return { namespace: ns, name };
            });
            deleteCronJobs(
              { items },
              {
                onSuccess: () => {
                  setShowBulkDeleteModal(false);
                  setSelectedCronJobIds(new Set());
                },
              }
            );
          }}
        />
      )}
    </div>
  );
};
