import {
  AnnotationBadge,
  ButtonGroup,
  LoadingSpinner,
  ResourceCell,
  ResourceDeletionButton,
  ResourceDetailDrawer,
  ResourceDetailDrawerHeader,
  ResourceDetailEmptyBody,
  ResourceLink,
  ResourceModificationButton,
  ResourceScaleButton,
  ScrollArea,
  SheetTitle,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
  TooltipProvider,
} from "@litelens/design-system";
import { FC, useEffect, useState } from "react";
import { useCatchForbiddenResources } from "../../../../../shared/hooks/async-events/useCatchForbiddenResources";
import { useMainLayoutContext } from "../../../../MainLayoutContext";
import { useDetailDrawerContext } from "../../../../shared/components/details/DetailDrawerContext";
import { ManagedFieldBlock } from "../../../../shared/components/ManagedFieldBlock";
import { useUnifiedTray } from "../../../../shared/components/trays/unified/UnifiedTrayContext";
import { EventsTable } from "../../../base/events/components/EventsTable";
import { useGetEvents } from "../../../base/events/hooks/data-access/useGetEvents";
import { PodStatusBadge } from "../../pods/components/PodStatusBadge";
import { useGetPods } from "../../pods/hooks/data-access/useGetPods";
import type { ReplicaSet } from "../api/resources";
import { useGetReplicaSetDetail } from "../hooks/data-access/useGetReplicaSetDetail";
import { useDeleteReplicaSet } from "../hooks/data-mutation/useDeleteReplicaSet";
import { useScaleReplicaSet } from "../hooks/data-mutation/useScaleReplicaSet";
import { ReplicaSetDeleteConfirmationModal } from "./ReplicaSetDeleteConfirmationModal";
import { ReplicaSetScaleModal } from "./ReplicaSetScaleModal";

const ReplicaSetDrawerCtaButtons: FC<{
  name: string;
  namespace: string;
  currentReplicas: number;
  isOwned: boolean;
  ownerKind: string;
  ownerName: string;
  onDeleted: () => void;
}> = ({ name, namespace, currentReplicas, isOwned, ownerKind, ownerName, onDeleted }) => {
  const { openTab } = useUnifiedTray();

  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [scaleOpen, setScaleOpen] = useState(false);
  const [scaleKey, setScaleKey] = useState(0);

  const { mutate: deleteMutate, isPending: isDeletePending } = useDeleteReplicaSet();
  const { mutate: scaleMutate, isPending: isScalePending } = useScaleReplicaSet();

  return (
    <>
      <ButtonGroup>
        <TooltipProvider>
          <ResourceScaleButton
            mode="icon-button"
            ariaLabel="Scale ReplicaSet"
            disabled={isScalePending}
            isNotAllowed={isOwned}
            notAllowedReason={`Owned by ${ownerKind} ${ownerName} — scale the parent resource instead.`}
            onClick={() => setScaleOpen(true)}
          />
          <ResourceModificationButton
            mode="icon-button"
            ariaLabel="Edit ReplicaSet"
            onClick={() => openTab("modification", { kind: "ReplicaSet", name, namespace })}
          />
          <ResourceDeletionButton
            mode="icon-button"
            ariaLabel="Delete ReplicaSet"
            disabled={isScalePending || isDeletePending}
            onClick={() => setShowDeleteModal(true)}
          />
        </TooltipProvider>
      </ButtonGroup>

      <ReplicaSetDeleteConfirmationModal
        open={showDeleteModal}
        mode="single"
        name={name}
        namespace={namespace}
        isPending={isDeletePending}
        onClose={() => setShowDeleteModal(false)}
        onConfirm={() => {
          deleteMutate({ namespace, name }, { onSuccess: () => onDeleted() });
        }}
      />

      <ReplicaSetScaleModal
        key={scaleKey}
        open={scaleOpen}
        name={name}
        currentReplicas={currentReplicas}
        isPending={isScalePending}
        onClose={() => {
          setScaleOpen(false);
          setScaleKey((k) => k + 1);
        }}
        onScale={(replicas) => {
          scaleMutate({ namespace, name, replicas });
          setScaleOpen(false);
          setScaleKey((k) => k + 1);
        }}
      />
    </>
  );
};

const ReplicaSetOverviewTab: FC<{ rs: ReplicaSet }> = ({ rs }) => {
  const { onToggleNamespaceDetail } = useDetailDrawerContext();
  return (
    <ScrollArea className="h-full">
      <div className="grid grid-cols-[160px_1fr] items-start gap-y-3 p-4">
        <span className="text-h3 text-muted-foreground">Created</span>
        <span className="text-body font-mono">
          {rs.Age} ago ({rs.CreatedAt})
        </span>

        <span className="text-h3 text-muted-foreground">Name</span>
        <span className="text-body font-mono">{rs.Name}</span>

        <span className="text-h3 text-muted-foreground">Namespace</span>
        <ResourceLink onClick={() => onToggleNamespaceDetail(rs.Namespace)}>
          {rs.Namespace}
        </ResourceLink>

        {Object.keys(rs.Labels ?? {}).length > 0 && (
          <>
            <span className="text-h3 text-muted-foreground">Labels</span>
            <div className="flex flex-wrap gap-1">
              {Object.entries(rs.Labels).map(([k, v]) => (
                <AnnotationBadge key={k} label={v ? `${k}=${v}` : k} />
              ))}
            </div>
          </>
        )}

        {Object.keys(rs.Annotations ?? {}).length > 0 && (
          <>
            <span className="text-h3 text-muted-foreground">Annotations</span>
            <div className="flex flex-wrap gap-1">
              {Object.entries(rs.Annotations).map(([k, v]) => (
                <AnnotationBadge key={k} label={v ? `${k}=${v}` : k} />
              ))}
            </div>
          </>
        )}

        {(rs.ManagedFields ?? []).length > 0 && (
          <>
            <span className="text-h3 text-muted-foreground self-start pt-0.5">Managed Fields</span>
            <div className="flex flex-col gap-2">
              {rs.ManagedFields.map((mf) => (
                <ManagedFieldBlock key={`${mf.Manager}/${mf.Operation}`} mf={mf} />
              ))}
            </div>
          </>
        )}

        {rs.OwnerKind && (
          <>
            <span className="text-h3 text-muted-foreground">Controlled By</span>
            <span className="text-body font-mono">
              {rs.OwnerKind} {rs.OwnerName}
            </span>
          </>
        )}

        {rs.Selector && (
          <>
            <span className="text-h3 text-muted-foreground">Selector</span>
            <span className="text-body font-mono">{rs.Selector}</span>
          </>
        )}

        {rs.NodeSelector && rs.NodeSelector !== "<none>" && (
          <>
            <span className="text-h3 text-muted-foreground">Node Selector</span>
            <span className="text-body font-mono">{rs.NodeSelector}</span>
          </>
        )}

        {(rs.Images ?? []).length > 0 && (
          <>
            <span className="text-h3 text-muted-foreground">Images</span>
            <div className="flex flex-col gap-0.5">
              {rs.Images.map((img, i) => (
                <span key={`${img}-${i}`} className="text-body break-all font-mono">
                  {img}
                </span>
              ))}
            </div>
          </>
        )}

        <span className="text-h3 text-muted-foreground">Replicas</span>
        <span className="text-body font-mono">{rs.ReplicasDetail}</span>

        <span className="text-h3 text-muted-foreground">Tolerations</span>
        <span className="text-body font-mono">{rs.Tolerations}</span>

        <span className="text-h3 text-muted-foreground">Affinities</span>
        <span className="text-body font-mono">{rs.Affinities}</span>

        {rs.PodStatus && (
          <>
            <span className="text-h3 text-muted-foreground">Pod Status</span>
            <span className="text-body font-mono">{rs.PodStatus}</span>
          </>
        )}
      </div>
    </ScrollArea>
  );
};

const ReplicaSetPodsTab: FC<{ rs: ReplicaSet }> = ({ rs }) => {
  const { activeContext } = useMainLayoutContext();
  const { onToggleNamespaceDetail, onTogglePodDetail } = useDetailDrawerContext();

  const { data: allPods = [] } = useGetPods({ context: activeContext, namespace: rs.Namespace });
  const pods = allPods
    .filter(
      (p) =>
        p.ControlledBy === "ReplicaSet" &&
        p.ControlledByName === rs.Name &&
        p.Namespace === rs.Namespace
    )
    .toSorted((a, b) => a.Name.localeCompare(b.Name));

  return (
    <ScrollArea className="h-full">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="text-xs">Name</TableHead>
            <TableHead className="text-xs">Namespace</TableHead>
            <TableHead className="text-xs">Ready</TableHead>
            <TableHead className="text-xs">CPU</TableHead>
            <TableHead className="text-xs">Memory</TableHead>
            <TableHead className="text-xs">Status</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {pods.length === 0 ? (
            <TableRow>
              <TableCell colSpan={6} className="text-muted-foreground py-12 text-center text-xs">
                Item list is empty
              </TableCell>
            </TableRow>
          ) : (
            pods.map((p) => (
              <TableRow key={`${p.Namespace}/${p.Name}`}>
                <TableCell className="max-w-40 truncate font-mono text-xs">
                  <ResourceLink
                    truncate
                    truncateTextClassName="max-w-40"
                    onClick={() => onTogglePodDetail(p.Namespace, p.Name)}
                  >
                    {p.Name}
                  </ResourceLink>
                </TableCell>
                <TableCell className="text-xs">
                  <ResourceLink truncate onClick={() => onToggleNamespaceDetail(p.Namespace)}>
                    {p.Namespace}
                  </ResourceLink>
                </TableCell>
                <TableCell className="text-xs">{p.Ready}</TableCell>
                <TableCell>
                  <ResourceCell label={p.CPU} percent={p.CPUPercent} />
                </TableCell>
                <TableCell>
                  <ResourceCell label={p.Memory} percent={p.MemPercent} />
                </TableCell>
                <TableCell>
                  <PodStatusBadge status={p.Status} />
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </ScrollArea>
  );
};

const ReplicaSetEventsTab: FC<{ rs: ReplicaSet }> = ({ rs }) => {
  const { activeContext } = useMainLayoutContext();

  const { data: events = [] } = useGetEvents({ context: activeContext, namespace: rs.Namespace });
  const rsEvents = events.filter(
    (e) =>
      e.InvolvedObjectKind.toLowerCase() === "replicaset" &&
      e.InvolvedObjectName === rs.Name &&
      e.Namespace === rs.Namespace
  );

  return (
    <ScrollArea className="h-full">
      <EventsTable events={rsEvents} />
    </ScrollArea>
  );
};

interface ReplicaSetDetailDrawerProps {
  rsName: string | null;
  rsNamespace: string | null;
  open: boolean;
  onClose: () => void;
}

const ReplicaSetDrawerBody: FC<
  ReplicaSetDetailDrawerProps & {
    rsName: string;
    rsNamespace: string;
    onDataChange: (rs: ReplicaSet | undefined) => void;
  }
> = ({ rsName, rsNamespace, open, onClose, onDataChange }) => {
  const { activeContext } = useMainLayoutContext();

  const { data: rs, isLoading } = useGetReplicaSetDetail(activeContext, rsNamespace, rsName);
  useCatchForbiddenResources("replicasets", {
    open,
    resourceName: rsName,
    resourceLabel: "ReplicaSet",
    onForbiddenDetected: onClose,
  });

  const [eventsVisible, setEventsVisible] = useState(false);

  useEffect(() => {
    onDataChange(rs);
  }, [rs, onDataChange]);

  if (isLoading) {
    return <LoadingSpinner className="h-auto flex-1" />;
  }

  if (!rs) {
    return <ResourceDetailEmptyBody resourceKind="ReplicaSet" />;
  }

  return (
    <Tabs
      defaultValue="overview"
      className="min-h-0 flex-1"
      onValueChange={(v) => {
        if (v === "events") setEventsVisible(true);
      }}
    >
      <TabsList className="w-full justify-start rounded-none border-b bg-transparent px-4">
        <TabsTrigger value="overview" className="text-xs">
          Overview
        </TabsTrigger>
        <TabsTrigger value="pods" className="text-xs">
          Pods
        </TabsTrigger>
        <TabsTrigger value="events" className="text-xs">
          Events
        </TabsTrigger>
      </TabsList>
      <TabsContent value="overview" className="mt-0 min-h-0 flex-1">
        <ReplicaSetOverviewTab rs={rs} />
      </TabsContent>
      <TabsContent value="pods" className="mt-0 min-h-0 flex-1">
        <ReplicaSetPodsTab rs={rs} />
      </TabsContent>
      <TabsContent value="events" className="mt-0 min-h-0 flex-1">
        {eventsVisible && <ReplicaSetEventsTab rs={rs} />}
      </TabsContent>
    </Tabs>
  );
};

export const ReplicaSetDetailDrawer: FC<ReplicaSetDetailDrawerProps> = ({
  rsName,
  rsNamespace,
  open,
  onClose,
}) => {
  const [rs, setRs] = useState<ReplicaSet | undefined>(undefined);

  const hasData = !!rsName && !!rsNamespace;

  return (
    <ResourceDetailDrawer open={open} onClose={onClose}>
      <ResourceDetailDrawerHeader>
        <SheetTitle className="text-h1">ReplicaSet: {rs?.Name ?? rsName}</SheetTitle>
        {rs && (
          <ReplicaSetDrawerCtaButtons
            name={rs.Name}
            namespace={rs.Namespace}
            currentReplicas={rs.Current}
            isOwned={rs.OwnerKind !== ""}
            ownerKind={rs.OwnerKind}
            ownerName={rs.OwnerName}
            onDeleted={onClose}
          />
        )}
      </ResourceDetailDrawerHeader>

      {hasData ? (
        <ReplicaSetDrawerBody
          key={`${rsName}/${rsNamespace}`}
          rsName={rsName}
          rsNamespace={rsNamespace}
          open={open}
          onClose={onClose}
          onDataChange={setRs}
        />
      ) : (
        <ResourceDetailEmptyBody resourceKind="ReplicaSet" />
      )}
    </ResourceDetailDrawer>
  );
};
