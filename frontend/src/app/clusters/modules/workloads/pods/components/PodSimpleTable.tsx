import {
  ResourceCell,
  ResourceLink,
  ScrollArea,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@litelens/design-system";
import { FC } from "react";
import { useDetailDrawerContext } from "../../../../shared/components/details/DetailDrawerContext";
import type { Pod } from "../api/resources";
import { PodStatusBadge } from "./PodStatusBadge";

interface PodSimpleTableProps {
  pods: Pod[];
}

export const PodSimpleTable: FC<PodSimpleTableProps> = ({ pods }) => {
  const { onToggleNamespaceDetail, onTogglePodDetail } = useDetailDrawerContext();

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
              <TableCell colSpan={6} className="py-12 text-center text-xs text-muted-foreground">
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
