func PrepareContainer(ctx context.Context, container ContainerContext, locales []string, dryRun bool, allLocalesRequired bool) (*StatusResult, error) {

	logger.Info(`Filling home page template`)

	homePage, err := RenderPage(ctx, home, container, locales, allLocalesRequired)
	if err != nil {
		return nil, err
	}

	templateIds := []string{homePage.ID}

	if container.PageSlugs.FAQ != "" {
		faqPage, err := RenderPage(ctx, faq, container, locales, allLocalesRequired)
		if err != nil {
			return nil, err
		}

		templateIds = append(templateIds, faqPage.ID)
	}

	if dryRun {
		return &StatusResult{Status: StatusDryRun}, nil
	}

	logger.Info(`Marking page template(s) for usage`, "template_ids", templateIds)

	if err := MarkReadyForUsage(ctx, container, templateIds); err != nil {
		return nil, err
	}

	return &StatusResult{Status: StatusComplete}, nil
}

func RenderPage(ctx context.Context, source Source, container ContainerContext, locales []string, allLocalesRequired bool) (string, error) {

	logger.Info(fmt.Sprintf(`Filling %s page template`, source.Name))

	template, err := FetchAndFillTemplate(ctx, source, container, locales)
	if err != nil {
		return nil, err
	}

	page, err := ConfigureFromTemplate(ctx, container, template, locales)
	if err != nil {
		return nil, err
	}

	if len(page.Locales) != len(locales) {
		const message = fmt.Sprintf(`Failed to render %s page template for some locales`, source.Name)
		if allLocalesRequired {
			return nil, fmt.Errorf(message)
		} else {
			logger.Warn(message, "locales", locales, "pages", page.Locales)
		}
	}

	return page, nil
}
