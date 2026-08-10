import {
  Checkbox,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
  EmptyState,
  ListChecksIcon,
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
import { FC, useState } from "react";
import { useGetJobs } from "./hooks/data-access/useGetJobs";
import { useDeleteJob } from "./hooks/data-mutation/useDeleteJob";
import { useDeleteJobs } from "./hooks/data-mutation/useDeleteJobs";
import { useMainLayoutContext } from "../../../MainLayoutContext";
import { useDetailDrawerContext } from "../../../shared/components/details/DetailDrawerContext";
import { useUnifiedTray } from "../../../shared/components/trays/unified/UnifiedTrayContext";
import { JobConditionBadge } from "./components/JobConditionBadge";
import { JobDeleteConfirmationModal } from "./components/JobDeleteConfirmationModal";

interface JobTableCtaButtonsProps {
  name: string;
  namespace: string;
}

const JobTableCtaButtons: FC<JobTableCtaButtonsProps> = ({ namespace, name }) => {
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const { openTab } = useUnifiedTray();

  const { mutate: deleteJob, isPending: isDeletePending } = useDeleteJob();

  const handleDeleteConfirm = () => {
    deleteJob({ namespace, name }, { onSuccess: () => setShowDeleteModal(false) });
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
            onClick={() => openTab("modification", { kind: "Job", name, namespace })}
          />
          <ResourceDeletionButton
            disabled={isDeletePending}
            onClick={() => setShowDeleteModal(true)}
          />
        </DropdownMenuContent>
      </DropdownMenu>

      <JobDeleteConfirmationModal
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

export const JobsView: FC = () => {
  const [search, setSearch] = useState("");
  const [selectedJobIds, setSelectedJobIds] = useState<Set<string>>(new Set());
  const [showBulkDeleteModal, setShowBulkDeleteModal] = useState(false);

  const { activeContext, namespace } = useMainLayoutContext();
  const { onToggleNamespaceDetail, onToggleJobDetail } = useDetailDrawerContext();

  const { mutate: deleteJobs, isPending: isBulkDeletePending } = useDeleteJobs();

  const { data: raw = [], isLoading } = useGetJobs({ context: activeContext, namespace });

  const jobs = raw
    .filter((j) => !search || j.Name.toLowerCase().includes(search.toLowerCase()))
    .toSorted((a, b) => a.Name.localeCompare(b.Name));

  return (
    <div className="flex h-full flex-col gap-3">
      <div className="flex items-center gap-3">
        <span className="text-h1">Jobs</span>
        <span className="text-muted-foreground text-xs">
          {jobs.length} item{jobs.length !== 1 ? "s" : ""}
        </span>
        <div className="ml-auto flex items-center gap-4">
          <ResourceBulkDeletionButton
            count={selectedJobIds.size}
            ariaLabel="Delete selected jobs"
            tooltip="Delete selected Jobs"
            onClick={() => setShowBulkDeleteModal(true)}
          />
          <SearchInput
            placeholder="Search Jobs..."
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
                  jobs.length > 0 &&
                  jobs.every((j) => selectedJobIds.has(`${j.Namespace}/${j.Name}`))
                }
                indeterminate={
                  jobs.some((j) => selectedJobIds.has(`${j.Namespace}/${j.Name}`)) &&
                  !jobs.every((j) => selectedJobIds.has(`${j.Namespace}/${j.Name}`))
                }
                onCheckedChange={(checked) => {
                  if (checked) {
                    const newSelection = new Set(selectedJobIds);
                    jobs.forEach((j) => newSelection.add(`${j.Namespace}/${j.Name}`));
                    setSelectedJobIds(newSelection);
                  } else {
                    const newSelection = new Set(selectedJobIds);
                    jobs.forEach((j) => newSelection.delete(`${j.Namespace}/${j.Name}`));
                    setSelectedJobIds(newSelection);
                  }
                }}
              />
            </TableHead>
            <TableHead>Name</TableHead>
            {!namespace && <TableHead>Namespace</TableHead>}
            <TableHead>Resumed</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Succeeded</TableHead>
            <TableHead>Completions</TableHead>
            <TableHead>Parallelism</TableHead>
            <TableHead>Duration</TableHead>
            <TableHead>Age</TableHead>
            <TableHead className="w-8" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            <TableSkeletonLoader
              rows={5}
              columns={namespace ? 7 : 8}
              includeCheckbox={true}
              columnWidths={[
                "w-[65%]",
                "w-[55%]",
                "w-[30%]",
                "w-[35%]",
                "w-[30%]",
                "w-[30%]",
                "w-[30%]",
                "w-[30%]",
              ]}
            />
          ) : jobs.length === 0 ? (
            <TableRow>
              <TableCell colSpan={namespace ? 10 : 11} className="px-0 py-0">
                <EmptyState
                  icon={<ListChecksIcon className="size-8" />}
                  title="No Jobs"
                  description="Create a Job to run a task to completion"
                />
              </TableCell>
            </TableRow>
          ) : (
            jobs.map((j) => (
              <TableRow
                key={`${j.Namespace}/${j.Name}`}
                className="cursor-pointer"
                onClick={() => onToggleJobDetail(j.Namespace, j.Name)}
              >
                <TableCell onClick={(e) => e.stopPropagation()}>
                  <Checkbox
                    checked={selectedJobIds.has(`${j.Namespace}/${j.Name}`)}
                    onCheckedChange={(checked) => {
                      const newSelection = new Set(selectedJobIds);
                      if (checked) {
                        newSelection.add(`${j.Namespace}/${j.Name}`);
                      } else {
                        newSelection.delete(`${j.Namespace}/${j.Name}`);
                      }
                      setSelectedJobIds(newSelection);
                    }}
                  />
                </TableCell>
                <TableCell className="font-mono text-xs">{j.Name}</TableCell>
                {!namespace && (
                  <TableCell className="text-xs">
                    <ResourceLink
                      onClick={(e) => {
                        e.stopPropagation();
                        onToggleNamespaceDetail(j.Namespace);
                      }}
                    >
                      {j.Namespace}
                    </ResourceLink>
                  </TableCell>
                )}
                <TableCell className="text-xs">{j.Resumed ? "True" : "False"}</TableCell>
                <TableCell>
                  <JobConditionBadge condition={j.Status} />
                </TableCell>
                <TableCell className="text-xs">{j.Succeeded}</TableCell>
                <TableCell className="text-xs">{j.Completions}</TableCell>
                <TableCell className="text-xs">{j.Parallelism}</TableCell>
                <TableCell className="text-xs">{j.Duration}</TableCell>
                <TableCell className="text-xs">{j.Age}</TableCell>
                <TableCell onClick={(e) => e.stopPropagation()}>
                  <JobTableCtaButtons name={j.Name} namespace={j.Namespace} />
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>

      {selectedJobIds.size > 0 && (
        <JobDeleteConfirmationModal
          open={showBulkDeleteModal}
          mode="bulk"
          items={Array.from(selectedJobIds).map((key) => {
            const [ns, name] = key.split("/");
            return { namespace: ns, name };
          })}
          isPending={isBulkDeletePending}
          onClose={() => setShowBulkDeleteModal(false)}
          onConfirm={() => {
            const items = Array.from(selectedJobIds).map((key) => {
              const [ns, name] = key.split("/");
              return { namespace: ns, name };
            });
            deleteJobs(
              { items },
              {
                onSuccess: () => {
                  setShowBulkDeleteModal(false);
                  setSelectedJobIds(new Set());
                },
              }
            );
          }}
        />
      )}
    </div>
  );
};
