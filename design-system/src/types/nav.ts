import { ElementType } from "react";

export interface NavItem<T> {
  id: string;
  label: string;
  view?: T;
}

export interface NavGroup<T> {
  id: string;
  label: string;
  icon: ElementType;
  items: NavItem<T>[];
}

export type NavEntry<T> =
  { kind: "item"; icon: ElementType; item: NavItem<T> } | { kind: "group"; group: NavGroup<T> };
