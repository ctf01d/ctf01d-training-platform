import { type ReactNode } from "react";

interface Column<T> {
  header: string;
  render: (item: T) => ReactNode;
  className?: string;
}

interface DataTableProps<T> {
  columns: Column<T>[];
  data: T[];
  loading?: boolean;
  emptyMessage?: string;
  page: number;
  perPage: number;
  total: number;
  onPageChange: (page: number) => void;
  actions?: (item: T) => ReactNode;
}

export function DataTable<T extends { id?: number }>({
  columns,
  data,
  loading,
  emptyMessage = "No data",
  page,
  perPage,
  total,
  onPageChange,
  actions,
}: DataTableProps<T>) {
  const totalPages = Math.ceil(total / perPage);

  if (loading) return <div className="loading">Loading...</div>;
  if (data.length === 0)
    return <div className="empty-state">{emptyMessage}</div>;

  return (
    <div className="table-shell">
      <table className="data-table">
        <thead>
          <tr>
            {columns.map((col) => (
              <th key={col.header} className={col.className}>
                {col.header}
              </th>
            ))}
            {actions && <th>Actions</th>}
          </tr>
        </thead>
        <tbody>
          {data.map((item, idx) => (
            <tr key={item.id ?? idx}>
              {columns.map((col) => (
                <td key={col.header} className={col.className}>
                  {col.render(item)}
                </td>
              ))}
              {actions && <td>{actions(item)}</td>}
            </tr>
          ))}
        </tbody>
      </table>
      {totalPages > 1 && (
        <div className="pagination">
          <button
            className="btn btn-sm"
            disabled={page <= 1}
            onClick={() => onPageChange(page - 1)}
          >
            Prev
          </button>
          <span>
            Page {page} of {totalPages} ({total} items)
          </span>
          <button
            className="btn btn-sm"
            disabled={page >= totalPages}
            onClick={() => onPageChange(page + 1)}
          >
            Next
          </button>
        </div>
      )}
    </div>
  );
}
