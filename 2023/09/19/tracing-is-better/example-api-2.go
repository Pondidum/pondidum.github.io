var tr = otel.Tracer("container_api")

func PrepareContainer(ctx context.Context, container ContainerContext, locales []string, dryRun bool, allLocalesRequired bool) (*StatusResult, error) {
	ctx, span := tr.Start(ctx, "prepare_container")
	defer span.End()

	homePage, err := RenderPage(ctx, home, container, locales, allLocalesRequired)
	if err != nil {
		return nil, tracing.Error(span, err)
	}

	templateIds := []string{homePage.ID}

	if container.PageSlugs.FAQ != "" {
		faqPage, err := RenderPage(ctx, faq, container, locales, allLocalesRequired)
		if err != nil {
			return nil, tracing.Error(span, err)
		}

		templateIds = append(templateIds, faqPage.ID)
	}

	if dryRun {
		return &StatusResult{Status: StatusDryRun}, nil
	}

	logger.Info(`Marking page template(s) for usage`, "template_ids", templateIds)

	if err := MarkReadyForUsage(ctx, container, templateIds); err != nil {
		return nil, tracing.Error(span, err)
	}

	return &StatusResult{Status: StatusComplete}, nil
}

func RenderPage(ctx context.Context, source Source, container ContainerContext, locales []string, allLocalesRequired bool) (string, error) {
	ctx, span := tr.Start(ctx, "render_page")
	defer span.End()

	logger.Info(fmt.Sprintf(`Filling %s page template`, source.Name))

	template, err := FetchAndFillTemplate(ctx, source, container, locales)
	if err != nil {
		return nil, tracing.Error(span, err)
	}

	page, err := ConfigureFromTemplate(ctx, container, template, locales)
	if err != nil {
		return nil, tracing.Error(span, err)
	}

	if len(page.Locales) != len(locales) {
		const message = fmt.Sprintf(`Failed to render %s page template for some locales`, source.Name)
		if allLocalesRequired {
			return nil, tracing.Errorf(message)
		} else {
			logger.Warn(message, "locales", locales, "pages", page.Locales)
		}
	}

	return page, nil
}
