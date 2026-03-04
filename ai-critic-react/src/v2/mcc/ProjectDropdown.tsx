import { ProjectChooser } from '../../components/chooser/ProjectChooser';
import type { ProjectChooserProps } from '../../components/chooser/ProjectChooser';

export type ProjectDropdownProps = ProjectChooserProps;

export function ProjectDropdown(props: ProjectDropdownProps) {
    return <ProjectChooser {...props} />;
}
