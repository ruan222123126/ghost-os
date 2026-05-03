import { OrchestrationEditorClient } from '@/components/orchestration/OrchestrationEditorClient';

interface OrchestrationEditorPageProps {
  params: {
    id: string;
  };
}

export default function OrchestrationEditorPage(props: OrchestrationEditorPageProps) {
  return <OrchestrationEditorClient orchestrationID={props.params.id} />;
}
