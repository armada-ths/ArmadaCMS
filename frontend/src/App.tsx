import { Admin, Resource } from "react-admin";
import { Layout } from "./Layout";
import { dataProvider } from "./dataProvider";
import { UserList } from "./components/User/UserList";
import { UserCreate } from "./components/User/UserCreate";
import { UserEdit } from "./components/User/UserEdit";
import { ProfileList } from "./components/Profile/ProfileList";
import { ProfileCreate } from "./components/Profile/ProfileCreate";
import { ProfileEdit } from "./components/Profile/ProfileEdit";
import { TeamList } from "./components/Team/TeamList";
import { TeamCreate } from "./components/Team/TeamCreate";
import { TeamEdit } from "./components/Team/TeamEdit";
import { authProvider } from "./context/authProvider";
import { TimelineList } from "./components/Timeline/TimelineList";
import { TimelineCreate } from "./components/Timeline/TimelineCreate";
import { TimelineEdit } from "./components/Timeline/TimelineEdit";
import { EmploymentCreate } from "./components/Employment/EmploymentCreate";
import { EmploymentEdit } from "./components/Employment/EmploymentEdit";
import { EmploymentList } from "./components/Employment/EmploymentList";
import { EventCreate } from "./components/Event/EventCreate";
import { EventEdit } from "./components/Event/EventEdit";
import { EventList } from "./components/Event/EventList";
import { ExhibitorCreate } from "./components/Exhibitor/ExhibitorCreate";
import { ExhibitorEdit } from "./components/Exhibitor/ExhibitorEdit";
import { ExhibitorList } from "./components/Exhibitor/ExhibitorList";
import { IndustryCreate } from "./components/Industry/IndustryCreate";
import { IndustryEdit } from "./components/Industry/IndustryEdit";
import { IndustryList } from "./components/Industry/IndustryList";
import { ProgramCreate } from "./components/Program/ProgramCreate";
import { ProgramEdit } from "./components/Program/ProgramEdit";
import { ProgramList } from "./components/Program/ProgramList";

export const App = () => (
  <Admin
    dataProvider={dataProvider}
    authProvider={authProvider}
    layout={Layout}
  >
    <Resource
      name="customusers"
      list={UserList}
      create={UserCreate}
      edit={UserEdit}
    />
    <Resource
      name="profiles"
      list={ProfileList}
      create={ProfileCreate}
      edit={ProfileEdit}
    />
    <Resource
      name="teams"
      list={TeamList}
      create={TeamCreate}
      edit={TeamEdit}
    />
    <Resource
      name="timeline"
      list={TimelineList}
      create={TimelineCreate}
      edit={TimelineEdit}
    />
    <Resource
      name="programs"
      list={ProgramList}
      create={ProgramCreate}
      edit={ProgramEdit}
    />
    <Resource
      name="industries"
      list={IndustryList}
      create={IndustryCreate}
      edit={IndustryEdit}
    />
    <Resource
      name="events"
      list={EventList}
      create={EventCreate}
      edit={EventEdit}
    />
    <Resource
      name="exhibitors"
      list={ExhibitorList}
      create={ExhibitorCreate}
      edit={ExhibitorEdit}
    />
    <Resource
      name="employments"
      list={EmploymentList}
      create={EmploymentCreate}
      edit={EmploymentEdit}
    />
  </Admin>
);
