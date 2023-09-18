var tr = otel.Tracer("container_api")

func PrepareContainer(ctx context.Context, container ContainerContext, locales []string, dryRun bool, allLocalesRequired bool) (*StatusResult, error) {
	ctx, span := tr.Start(ctx, "prepare_container")
	defer span.End()

	tracing.StringSlice(span, "locales", locales)
	tracing.Bool(span, "dry_run", dryRun)
	tracing.Bool(span, "locales_mandatory", allLocalesRequired)

	homePage, err := RenderPage(ctx, home, container, locales, allLocalesRequired)
	if err != nil {
		return nil, tracing.Error(span, err)
	}

	templateIds := []string{homePage.ID}

	hasFaq := container.PageSlugs.FAQ != ""
	tracing.Bool(span, "has_faq", hasFaq)

	if hasFaq {
		faqPage, err := RenderPage(ctx, faq, container, locales, allLocalesRequired)
		if err != nil {
			return nil, tracing.Error(span, err)
		}

		templateIds = append(templateIds, faqPage.ID)
	}

	tracing.StringSlice(span, "template_ids", templateIds)

	if dryRun {
		return &StatusResult{Status: StatusDryRun}, nil
	}

	if err := MarkReadyForUsage(ctx, container, templateIds); err != nil {
		return nil, tracing.Error(span, err)
	}

	return &StatusResult{Status: StatusComplete}, nil
}

func RenderPage(ctx context.Context, source Source, container ContainerContext, locales []string, allLocalesRequired bool) (string, error) {
	ctx, span := tr.Start(ctx, "render_page")
	defer span.End()

	tracing.String(span, "source_name", source.Name)

	template, err := FetchAndFillTemplate(ctx, source, container, locales)
	if err != nil {
		return nil, tracing.Error(span, err)
	}

	page, err := ConfigureFromTemplate(ctx, container, template, locales)
	if err != nil {
		return nil, tracing.Error(span, err)
	}

	allLocalesRendered := len(page.Locales) == len(locales)

	tracing.Bool(span, "all_locales_rendered", allLocalesRendered)
	tracing.StringSlice(span, "locales_rendered", page.Locales)

	if !allLocalesRendered && required {
		return nil, tracing.Errorf(`Failed to render %s page template for some locales`, source.Name)
	}

	return page, nil
}
