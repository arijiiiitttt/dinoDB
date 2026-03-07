package storage

const PageSize = 4096 

type Page struct {
	ID   uint64          
	Data [PageSize]byte  
}

func NewPage(id uint64) *Page {
	return &Page{ID: id}
}
