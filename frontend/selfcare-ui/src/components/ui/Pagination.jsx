export default function Pagination({ page, totalPages, onPageChange, total }) {
  if (totalPages <= 1) return null

  return (
    <div className="flex items-center justify-between bg-white px-4 py-3 rounded-lg shadow">
      <div className="text-sm text-gray-500">
        Showing page {page} of {totalPages} ({total} total)
      </div>
      <div className="flex gap-2">
        <button
          onClick={() => onPageChange(page - 1)}
          disabled={page === 1}
          className="px-3 py-1 rounded border bg-white hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          Previous
        </button>
        <button
          onClick={() => onPageChange(page + 1)}
          disabled={page === totalPages}
          className="px-3 py-1 rounded border bg-white hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          Next
        </button>
      </div>
    </div>
  )
}
