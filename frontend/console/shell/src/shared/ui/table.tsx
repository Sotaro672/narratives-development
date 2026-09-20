// frontend/console/shell/src/shared/ui/table.tsx

import "./table.css";

function cn(...classes: Array<string | false | null | undefined>) {
  return classes.filter(Boolean).join(" ");
}

/**
 * Note on typings:
 * The project setup currently doesn't expose the global `JSX` namespace,
 * so we avoid React-specific or JSX intrinsic element typings here.
 * Props are typed with Record<string, any> to keep DX simple and avoid TS2503.
 */

type WithClassName = {
  className?: string;
} & Record<string, any>;

type TableAlign = "left" | "center" | "right";

type TableRowProps = WithClassName & {
  interactive?: boolean;
};

type TableHeadProps = WithClassName & {
  align?: TableAlign;
};

type TableCellProps = WithClassName & {
  align?: TableAlign;
};

export function Table(props: WithClassName) {
  const { className, ...rest } = props;

  return (
    <div data-slot="table-container" className="table__container">
      <table
        data-slot="table"
        className={cn("table", className)}
        {...rest}
      />
    </div>
  );
}

export function TableHeader(props: WithClassName) {
  const { className, ...rest } = props;

  return (
    <thead
      data-slot="table-header"
      className={cn("table__header", className)}
      {...rest}
    />
  );
}

export function TableBody(props: WithClassName) {
  const { className, ...rest } = props;

  return (
    <tbody
      data-slot="table-body"
      className={cn("table__body", className)}
      {...rest}
    />
  );
}

export function TableFooter(props: WithClassName) {
  const { className, ...rest } = props;

  return (
    <tfoot
      data-slot="table-footer"
      className={cn("table__footer", className)}
      {...rest}
    />
  );
}

export function TableRow(props: TableRowProps) {
  const {
    className,
    interactive = false,
    ...rest
  } = props;

  return (
    <tr
      data-slot="table-row"
      className={cn(
        "table__row",
        interactive && "table__row--interactive",
        className,
      )}
      {...rest}
    />
  );
}

export function TableHead(props: TableHeadProps) {
  const {
    className,
    align = "left",
    ...rest
  } = props;

  return (
    <th
      data-slot="table-head"
      className={cn(
        "table__head",
        `table__head--${align}`,
        className,
      )}
      {...rest}
    />
  );
}

export function TableCell(props: TableCellProps) {
  const {
    className,
    align = "left",
    ...rest
  } = props;

  return (
    <td
      data-slot="table-cell"
      className={cn(
        "table__cell",
        `table__cell--${align}`,
        className,
      )}
      {...rest}
    />
  );
}

export function TableCaption(props: WithClassName) {
  const { className, ...rest } = props;

  return (
    <caption
      data-slot="table-caption"
      className={cn("table__caption", className)}
      {...rest}
    />
  );
}