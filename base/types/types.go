package types

// IDRequest is used for URI-based ID extraction.
type IDRequest struct {
	ID int64 `uri:"id" binding:"required"`
}

// Pagination holds common pagination parameters.
type Pagination struct {
	Page     int `form:"page,default=1" json:"page" binding:"omitempty,min=1"`
	PageSize int `form:"pageSize,default=20" json:"pageSize" binding:"omitempty,min=1,max=100"`
}

// ListResponse wraps paginated list results.
type ListResponse struct {
	*Pagination
	Total int64 `json:"total"`
}
