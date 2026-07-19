package domain

import (
	"encoding/json"
	"time"
	"unicode/utf8"

	"github.com/trvux/elc-go/internal/platform/apperr"
	"github.com/trvux/elc-go/internal/platform/media"
	"github.com/trvux/elc-go/internal/platform/seo"
)

// ImageAsset re-exports the shared media type — see product/domain/types.go's
// identical alias for why this is centralized rather than duplicated.
type ImageAsset = media.ImageAsset

type Project struct {
	id                string
	title             string
	slug              string
	description       json.RawMessage
	images            []ImageAsset
	isFeatured        bool
	isPublished       bool
	metaTitle         *string
	metaDescription   *string
	orderIndex        int
	projectTypeID     *string
	clientName        string
	location          string
	completedAt       *time.Time
	testimonialQuote  string
	testimonialAuthor string
	createdAt         time.Time
	updatedAt         time.Time
	deletedAt         *time.Time
}

// ProjectTypeRef/CategoryGroupRef/ServiceGroupRef/ServiceRef are lightweight,
// read-only references to entities owned by other modules (project-type,
// group, service). project-type has NOT been migrated to Go yet — this
// module only ever reads project_type via a LEFT JOIN in its own SQL, never
// writes it, same pattern internal/service uses for GroupRef/CategoryRef
// (see internal/service/domain/types.go).
type ProjectTypeRef struct {
	ID   string
	Name string
	Slug string
}

// CategoryGroupRef mirrors exactly what the old TS mapToDomainWithCategory
// read off category.group_categories for a project's category: id+name only
// (no slug) — see modules/project/infrastructure/projectRepo.ts.
type CategoryGroupRef struct {
	ID   string
	Name string
}

// ServiceGroupRef mirrors what the old TS read off service.group for a
// project's service: id+name+slug (unlike CategoryGroupRef above).
type ServiceGroupRef struct {
	ID   string
	Name string
	Slug string
}

// ProjectCategory is one row of the project_category join — a category
// attached to a project under a specific condition (new/used).
type ProjectCategory struct {
	ID        string
	Name      string
	Slug      string
	GroupID   *string
	Condition string
	Group     *CategoryGroupRef
}

// ProjectServiceRef is one row of the project_service join.
type ProjectServiceRef struct {
	ID    string
	Title string
	Slug  string
	Group *ServiceGroupRef
}

// ProjectWithRelations is what read queries (GetAll/GetByID/GetBySlug)
// return — a Project plus its joined project_type/categories/services.
// Create/Update only ever deal with a plain *Project; the relations
// (categories with condition, service ids) are passed as separate
// parameters — see repository.go's CategoryCondition and Create/Update
// signatures.
type ProjectWithRelations struct {
	*Project
	ProjectType *ProjectTypeRef
	Categories  []ProjectCategory
	Services    []ProjectServiceRef
	Tags        []TagRef
}

// TagRef is a lightweight read-only reference to a tag owned by the tag
// module — resolved via a direct SQL join into `tags`/`project_tags`, same
// cross-module read pattern as CategoryGroupRef/ServiceGroupRef above.
type TagRef struct {
	ID   string
	Name string
	Slug string
}

// CategoryCondition is one entry of the project_category join table's
// payload at write time: a category id plus the condition (new/used) it's
// attached under. condition is modeled as a plain string + validation, same
// choice internal/product/domain/types.go made for the identical
// product_condition Postgres enum — not a Go enum type.
type CategoryCondition struct {
	CategoryID string
	Condition  string
}

// AdjacentProject is the prev/next navigation shape used by GetAdjacent.
type AdjacentProject struct {
	Title string
	Slug  string
}

// NewProject validates and creates a new entity from user input.
func NewProject(
	title, slug string,
	description json.RawMessage,
	images []ImageAsset,
	isFeatured, isPublished bool,
	metaTitle, metaDescription *string,
	orderIndex int,
	projectTypeID *string,
	clientName, location string,
	completedAt *time.Time,
	testimonialQuote, testimonialAuthor string,
) (*Project, error) {
	fields := map[string][]string{}

	if errs := validateTitle(title); len(errs) > 0 {
		fields["title"] = errs
	}
	if errs := validateSlug(slug); len(errs) > 0 {
		fields["slug"] = errs
	}
	if errs := seo.ValidateMetaTitle(metaTitle); len(errs) > 0 {
		fields["metaTitle"] = errs
	}
	if errs := seo.ValidateMetaDescription(metaDescription); len(errs) > 0 {
		fields["metaDescription"] = errs
	}

	if len(fields) > 0 {
		return nil, apperr.NewValidationError("validation failed", fields)
	}

	if len(description) == 0 {
		description = json.RawMessage(`{}`)
	}
	if images == nil {
		images = []ImageAsset{}
	}

	now := time.Now()
	return &Project{
		title:             title,
		slug:              slug,
		description:       description,
		images:            images,
		isFeatured:        isFeatured,
		isPublished:       isPublished,
		metaTitle:         metaTitle,
		metaDescription:   metaDescription,
		orderIndex:        orderIndex,
		projectTypeID:     projectTypeID,
		clientName:        clientName,
		location:          location,
		completedAt:       completedAt,
		testimonialQuote:  testimonialQuote,
		testimonialAuthor: testimonialAuthor,
		createdAt:         now,
		updatedAt:         now,
	}, nil
}

// RehydrateProject reconstructs from a trusted DB row — no validation. Only
// the infrastructure layer should call this.
func RehydrateProject(
	id, title, slug string,
	description json.RawMessage,
	images []ImageAsset,
	isFeatured, isPublished bool,
	metaTitle, metaDescription *string,
	orderIndex int,
	projectTypeID *string,
	clientName, location string,
	completedAt *time.Time,
	testimonialQuote, testimonialAuthor string,
	createdAt, updatedAt time.Time,
	deletedAt *time.Time,
) *Project {
	return &Project{
		id:                id,
		title:             title,
		slug:              slug,
		description:       description,
		images:            images,
		isFeatured:        isFeatured,
		isPublished:       isPublished,
		metaTitle:         metaTitle,
		metaDescription:   metaDescription,
		orderIndex:        orderIndex,
		projectTypeID:     projectTypeID,
		clientName:        clientName,
		location:          location,
		completedAt:       completedAt,
		testimonialQuote:  testimonialQuote,
		testimonialAuthor: testimonialAuthor,
		createdAt:         createdAt,
		updatedAt:         updatedAt,
		deletedAt:         deletedAt,
	}
}

func (p *Project) ID() string                   { return p.id }
func (p *Project) Title() string                { return p.title }
func (p *Project) Slug() string                 { return p.slug }
func (p *Project) Description() json.RawMessage { return p.description }
func (p *Project) Images() []ImageAsset         { return p.images }
func (p *Project) IsFeatured() bool             { return p.isFeatured }
func (p *Project) IsPublished() bool            { return p.isPublished }
func (p *Project) MetaTitle() *string           { return p.metaTitle }
func (p *Project) MetaDescription() *string     { return p.metaDescription }
func (p *Project) OrderIndex() int              { return p.orderIndex }
func (p *Project) ProjectTypeID() *string       { return p.projectTypeID }
func (p *Project) ClientName() string           { return p.clientName }
func (p *Project) Location() string             { return p.location }
func (p *Project) CompletedAt() *time.Time      { return p.completedAt }
func (p *Project) TestimonialQuote() string     { return p.testimonialQuote }
func (p *Project) TestimonialAuthor() string    { return p.testimonialAuthor }
func (p *Project) CreatedAt() time.Time         { return p.createdAt }
func (p *Project) UpdatedAt() time.Time         { return p.updatedAt }
func (p *Project) DeletedAt() *time.Time        { return p.deletedAt }

func (p *Project) IsDeleted() bool {
	return p.deletedAt != nil
}

func (p *Project) UpdateTitle(title string) error {
	if errs := validateTitle(title); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"title": errs})
	}
	p.title = title
	p.updatedAt = time.Now()
	return nil
}

func (p *Project) UpdateSlug(slug string) error {
	if errs := validateSlug(slug); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"slug": errs})
	}
	p.slug = slug
	p.updatedAt = time.Now()
	return nil
}

func (p *Project) UpdateDescription(description json.RawMessage) {
	if len(description) == 0 {
		description = json.RawMessage(`{}`)
	}
	p.description = description
	p.updatedAt = time.Now()
}

func (p *Project) UpdateImages(images []ImageAsset) {
	if images == nil {
		images = []ImageAsset{}
	}
	p.images = images
	p.updatedAt = time.Now()
}

func (p *Project) SetFeatured(isFeatured bool) {
	p.isFeatured = isFeatured
	p.updatedAt = time.Now()
}

func (p *Project) SetPublished(isPublished bool) {
	p.isPublished = isPublished
	p.updatedAt = time.Now()
}

func (p *Project) UpdateMetaTitle(metaTitle *string) error {
	if errs := seo.ValidateMetaTitle(metaTitle); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"metaTitle": errs})
	}
	p.metaTitle = metaTitle
	p.updatedAt = time.Now()
	return nil
}

func (p *Project) UpdateMetaDescription(metaDescription *string) error {
	if errs := seo.ValidateMetaDescription(metaDescription); len(errs) > 0 {
		return apperr.NewValidationError("validation failed", map[string][]string{"metaDescription": errs})
	}
	p.metaDescription = metaDescription
	p.updatedAt = time.Now()
	return nil
}

func (p *Project) Reorder(orderIndex int) {
	p.orderIndex = orderIndex
	p.updatedAt = time.Now()
}

func (p *Project) UpdateProjectTypeID(projectTypeID *string) {
	p.projectTypeID = projectTypeID
	p.updatedAt = time.Now()
}

func (p *Project) UpdateClientName(clientName string) {
	p.clientName = clientName
	p.updatedAt = time.Now()
}

func (p *Project) UpdateLocation(location string) {
	p.location = location
	p.updatedAt = time.Now()
}

func (p *Project) UpdateCompletedAt(completedAt *time.Time) {
	p.completedAt = completedAt
	p.updatedAt = time.Now()
}

func (p *Project) UpdateTestimonial(quote, author string) {
	p.testimonialQuote = quote
	p.testimonialAuthor = author
	p.updatedAt = time.Now()
}

func (p *Project) MarkDeleted(deletedAt time.Time) {
	p.deletedAt = &deletedAt
}

func (p *Project) Restore() {
	p.deletedAt = nil
	p.updatedAt = time.Now()
}

func validateTitle(title string) []string {
	var errs []string
	if title == "" {
		errs = append(errs, "title is required")
	} else if utf8.RuneCountInString(title) > 200 {
		errs = append(errs, "title must not exceed 200 characters")
	}
	return errs
}

func validateSlug(slug string) []string {
	var errs []string
	if slug == "" {
		errs = append(errs, "slug is required")
	} else if utf8.RuneCountInString(slug) > 200 {
		errs = append(errs, "slug must not exceed 200 characters")
	}
	return errs
}

// ValidateCondition enforces the product_condition Postgres enum's two
// allowed values — same rule internal/product/domain/types.go's
// validateCondition uses for the identical enum.
func ValidateCondition(condition string) []string {
	if condition != "new" && condition != "used" {
		return []string{"condition must be 'new' or 'used'"}
	}
	return nil
}

type CreateProjectInput struct {
	Title             string
	Slug              string
	Description       json.RawMessage
	Images            []ImageAsset
	IsFeatured        bool
	IsPublished       bool
	MetaTitle         *string
	MetaDescription   *string
	OrderIndex        int
	ProjectTypeID     *string
	ServiceIDs        []string
	Categories        []CategoryCondition
	TagIDs            []string
	ClientName        string
	Location          string
	CompletedAt       *time.Time
	TestimonialQuote  string
	TestimonialAuthor string
}

type UpdateProjectInput struct {
	ID              string
	Title           *string
	Slug            *string
	Description     json.RawMessage
	Images          []ImageAsset
	IsFeatured      *bool
	IsPublished     *bool
	MetaTitle       *string
	MetaDescription *string
	OrderIndex      *int
	ProjectTypeID   *string
	// ServiceIDs/Categories: nil means "leave relations untouched", a
	// non-nil pointer (including one pointing at an empty slice) means
	// "replace all relations with this set" — mirrors the TS repository's
	// `if (input.serviceIds !== undefined)` / `if (input.categories !==
	// undefined)` distinction between "field omitted" and "field sent
	// empty" exactly (see modules/project/infrastructure/projectRepo.ts).
	ServiceIDs        *[]string
	Categories        *[]CategoryCondition
	TagIDs            *[]string
	ClientName        *string
	Location          *string
	CompletedAt       *time.Time
	TestimonialQuote  *string
	TestimonialAuthor *string
}

type ProjectFilter struct {
	ProjectTypeID  *string
	CategorySlug   *string
	CategorySlugs  []string
	ServiceSlug    *string
	ServiceSlugs   []string
	ExcludeID      *string
	IsPublished    *bool
	IsFeatured     *bool
	Search         string
	Limit          int
	Offset         int
	IncludeDeleted bool
	OrderBy        string // "orderIndex" | "createdAt" | "title"
	OrderDirection string // "asc" | "desc"
}

// CategoryRef is the {id, name, slug} shape returned by
// GetCategoriesByProjectTypeID — deliberately not domain.ProjectCategory
// (that carries condition/group/pricing this query never needs).
type CategoryRef struct {
	ID   string
	Name string
	Slug string
}
