package tools

func (t *Tool) DisplayName() string {
	title := t.Function.Annotations.Title
	if title != "" {
		return title
	}

	return t.Function.Name
}
