// frontend/console/shell/src/shared/ui/table.tsx

import "./table.css";

/** Simple className merger */
function cn(...classes: Array<string | false | null | undefined>) {
  return classes.filter(Boolean).join(" ");
}

/**
 * Note on typings:
 * The project setup currently doesn't expose the global `JSX` namespace,
 * so we avoid React-specific or JSX intrinsic element typings here.
 * Props are typed as `any` with a `className?: string` pick to keep DX decent
 * and eliminate TS2503 errors.
 */

type WithClassName = {
  className?: string;
} & Record<string, any>;

export function Table(props: WithClassName) {
  const {
    className,
    ...rest
  } = props;

  return (
    <div
      data-slot="table-container"
      className="table__container"
    >
      <table
        data-slot="table"
        className={cn(
          "table",
          className,
        )}
        {...rest}
      />
    </div>
  );
}

export function TableHeader(props: WithClassName) {
  const {
    className,
    ...rest
  } = props;

  return (
    <thead
      data-slot="table-header"
      className={cn(
        "table__header",
        className,
      )}
      {...rest}
    />
  );
}

export function TableBody(props: WithClassName) {
  const {
    className,
    ...rest
  } = props;

  return (
    <tbody
      data-slot="table-body"
      className={cn(
        "table__body",
        className,
      )}
      {...rest}
    />
  );
}

export function TableFooter(props: WithClassName) {
  const {
    className,
    ...rest
  } = props;

  return (
    <tfoot
      data-slot="table-footer"
      className={cn(
        "table__footer",
        className,
      )}
      {...rest}
    />
  );
}

export function TableRow(props: WithClassName) {
  const {
    className,
    ...rest
  } = props;

  return (
    <tr
      data-slot="table-row"
      className={cn(
        "table__row",
        className,
      )}
      {...rest}
    />
  );
}

export function TableHead(props: WithClassName) {
  const {
    className,
    ...rest
  } = props;

  return (
    <th
      data-slot="table-head"
      className={cn(
        "table__head",
        className,
      )}
      {...rest}
    />
  );
}

export function TableCell(props: WithClassName) {
  const {
    className,
    ...rest
  } = props;

  return (
    <td
      data-slot="table-cell"
      className={cn(
        "table__cell",
        className,
      )}
      {...rest}
    />
  );
}

export function TableCaption(props: WithClassName) {
  const {
    className,
    ...rest
  } = props;

  return (
    <caption
      data-slot="table-caption"
      className={cn(
        "table__caption",
        className,
      )}
      {...rest}
    />
  );
}