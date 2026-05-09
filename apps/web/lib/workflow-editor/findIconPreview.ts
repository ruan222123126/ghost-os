const FIND_ICON_TEMPLATE_PREVIEW_ENDPOINT = '/api/tools/screen/find-icon/template-preview';

export function buildFindIconTemplatePreviewURL(templatePath: string): string {
  const path = templatePath.trim();
  if (!path) {
    return '';
  }
  return `${FIND_ICON_TEMPLATE_PREVIEW_ENDPOINT}?template_path=${encodeURIComponent(path)}`;
}

