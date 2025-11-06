// frontend/shared/ui/table.tsx
// Standalone Table UI primitives (no external imports, no JSX namespace types)

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

type WithClassName = { className?: string } & Record<string, any>;

export function Table(props: WithClassName) {
  const { className, ...rest } = props;
  return (
    <div data-slot="table-container" className="relative w-full overflow-x-auto">
      <table
        data-slot="table"
        className={cn("w-full caption-bottom text-sm", className)}
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
      className={cn("[&_tr]:border-b", className)}
      {...rest}
    />
  );
}

export function TableBody(props: WithClassName) {
  const { className, ...rest } = props;
  return (
    <tbody
      data-slot="table-body"
      className={cn("[&_tr:last-child]:border-0", className)}
      {...rest}
    />
  );
}

export function TableFooter(props: WithClassName) {
  const { className, ...rest } = props;
  return (
    <tfoot
      data-slot="table-footer"
      className={cn(
        "bg-muted/50 border-t font-medium [&>tr]:last:border-b-0",
        className
      )}
      {...rest}
    />
  );
}

export function TableRow(props: WithClassName) {
  const { className, ...rest } = props;
  return (
    <tr
      data-slot="table-row"
      className={cn(
        "hover:bg-muted/50 data-[state=selected]:bg-muted border-b transition-colors",
        className
      )}
      {...rest}
    />
  );
}

export function TableHead(props: WithClassName) {
  const { className, ...rest } = props;
  return (
    <th
      data-slot="table-head"
      className={cn(
        "text-foreground h-10 px-2 text-left align-middle font-medium whitespace-nowrap " +
          "[&:has([role=checkbox])]:pr-0 [&>[role=checkbox]]:translate-y-[2px]",
        className
      )}
      {...rest}
    />
  );
}

export function TableCell(props: WithClassName) {
  const { className, ...rest } = props;
  return (
    <td
      data-slot="table-cell"
      className={cn(
        "p-2 align-middle whitespace-nowrap " +
          "[&:has([role=checkbox])]:pr-0 [&>[role=checkbox]]:translate-y-[2px]",
        className
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
      className={cn("text-muted-foreground mt-4 text-sm", className)}
      {...rest}
    />
  );
}
