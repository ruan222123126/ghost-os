import { WorkflowEditorClient } from '@/components/workflow/WorkflowEditorClient';

interface WorkflowTaskPageProps {
  params: {
    id: string;
  };
}

export default function WorkflowTaskPage(props: WorkflowTaskPageProps) {
  const { id } = props.params;

  return <WorkflowEditorClient mode="edit" taskID={id} />;
}
