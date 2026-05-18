import {
  Admin,
  AppBar,
  Logout,
  Menu,
  Resource,
  Layout,
  UserMenu,
} from "react-admin";
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
import { FairDateList } from "./components/FairDate/FairDateList";
import { FairDateCreate } from "./components/FairDate/FairDateCreate";
import { FairDateEdit } from "./components/FairDate/FairDateEdit";
import { FeatureFlagList } from "./components/FeatureFlag/FeatureFlagList";
import { FeatureFlagEdit } from "./components/FeatureFlag/FeatureFlagEdit";
import { HighlightCardList } from "./components/HighlightCard/HighlightCardList";
import { HighlightCardCreate } from "./components/HighlightCard/HighlightCardCreate";
import { HighlightCardEdit } from "./components/HighlightCard/HighlightCardEdit";
import { RoleList } from "./components/Role/RoleList";
import { RoleCreate } from "./components/Role/RoleCreate";
import { RoleEdit } from "./components/Role/RoleEdit";
import { RecruitmentPeriodList } from "./components/RecruitmentPeriod/RecruitmentPeriodList";
import { RecruitmentPeriodCreate } from "./components/RecruitmentPeriod/RecruitmentPeriodCreate";
import { RecruitmentPeriodEdit } from "./components/RecruitmentPeriod/RecruitmentPeriodEdit";
import { RecruitmentRoleList } from "./components/RecruitmentRole/RecruitmentRoleList";
import { RecruitmentRoleCreate } from "./components/RecruitmentRole/RecruitmentRoleCreate";
import { RecruitmentRoleEdit } from "./components/RecruitmentRole/RecruitmentRoleEdit";
import { AuditLogList } from "./components/AuditLog/AuditLogList";
import { AuditLogShow } from "./components/AuditLog/AuditLogShow";
import { CustomLoginPage } from "./components/CustomLoginPage";
import { BlogpostList } from "./components/Blogpost/BlogpostList";
import { BlogpostCreate } from "./components/Blogpost/BlogpostCreate";
import { BlogpostEdit } from "./components/Blogpost/BlogpostEdit";
import { ChangePasswordButton } from "./components/ChangePasswordButton";
import { Dashboard } from "./components/Dashboard/Dashboard";
import { DashboardMenuItem } from "react-admin";
import { ReactNode } from "react";

export const MyMenu = () => (
  <Menu>
    <DashboardMenuItem primaryText="Dashboard" />
    <Menu.ResourceItems />
  </Menu>
);

const MyUserMenu = () => (
  <UserMenu>
    <ChangePasswordButton />
    <Logout />
  </UserMenu>
);

const MyAppBar = () => <AppBar userMenu={<MyUserMenu />} />;

export const MyLayout = ({ children }: { children?: ReactNode }) => (
  <Layout menu={MyMenu} appBar={MyAppBar}>
    {children}
  </Layout>
);

export const App = () => (
  <Admin
    basename="/admin"
    dataProvider={dataProvider}
    authProvider={authProvider}
    layout={MyLayout}
    loginPage={CustomLoginPage}
    dashboard={Dashboard}
  >
    <Resource
      name="customusers"
      options={{ label: "Custom users" }}
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
    <Resource
      name="fairdates"
      options={{ label: "Fair dates" }}
      list={FairDateList}
      create={FairDateCreate}
      edit={FairDateEdit}
    />
    <Resource
      name="featureflags"
      options={{ label: "Feature flags" }}
      list={FeatureFlagList}
      edit={FeatureFlagEdit}
    />

    <Resource
      name="highlightcards"
      options={{ label: "Highlight cards" }}
      list={HighlightCardList}
      create={HighlightCardCreate}
      edit={HighlightCardEdit}
    />

    <Resource
      name="roles"
      list={RoleList}
      create={RoleCreate}
      edit={RoleEdit}
    />

    <Resource
      name="recruitmentperiods"
      options={{ label: "Recruitment periods" }}
      list={RecruitmentPeriodList}
      create={RecruitmentPeriodCreate}
      edit={RecruitmentPeriodEdit}
    />

    <Resource
      name="recruitmentroles"
      options={{ label: "Recruitment roles" }}
      list={RecruitmentRoleList}
      create={RecruitmentRoleCreate}
      edit={RecruitmentRoleEdit}
    />

    <Resource
      name="auditlogs"
      options={{ label: "Audit logs" }}
      list={AuditLogList}
      show={AuditLogShow}
    />

    <Resource
      name="blogposts"
      options={{ label: "Blog posts" }}
      list={BlogpostList}
      create={BlogpostCreate}
      edit={BlogpostEdit}
    />
  </Admin>
);
