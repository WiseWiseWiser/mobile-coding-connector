interface PaginationProps {
    currentPage: number;
    totalPages: number;
    totalCount: number;
    pageSize: number;
    onPageChange: (page: number) => void;
    loading?: boolean;
}

export function Pagination({
    currentPage,
    totalPages,
    totalCount,
    pageSize,
    onPageChange,
    loading = false,
}: PaginationProps) {
    if (totalPages <= 1) return null;

    const startItem = (currentPage - 1) * pageSize + 1;
    const endItem = Math.min(currentPage * pageSize, totalCount);

    return (
        <div className="mcc-agent-pagination">
            <div className="mcc-agent-pagination-info">
                Showing {startItem}-{endItem} of {totalCount} sessions
            </div>
            <div className="mcc-agent-pagination-controls">
                <button
                    className="mcc-agent-pagination-btn"
                    onClick={() => onPageChange(currentPage - 1)}
                    disabled={currentPage === 1 || loading}
                >
                    ←
                </button>
                <span className="mcc-agent-pagination-page">
                    Page {currentPage} of {totalPages}
                </span>
                <button
                    className="mcc-agent-pagination-btn"
                    onClick={() => onPageChange(currentPage + 1)}
                    disabled={currentPage === totalPages || loading}
                >
                    →
                </button>
            </div>
        </div>
    );
}
