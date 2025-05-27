import { useState, useMemo } from "react";
import { defineRouteConfig } from "@medusajs/admin-sdk";
import {
  createDataTableColumnHelper,
  useDataTable,
  DataTable,
  Badge,
  Button,
  Heading,
  Container,
  Text,
} from "@medusajs/ui";
import {
  Plus as PlusIcon,
  EllipsisHorizontal as EllipsisHorizontalIcon,
  Pencil,
} from "@medusajs/icons";
import { ActionMenu } from "../../../components/action-menu";

// Mock data structure based on your Plan model
interface Plan {
  id: string;
  name: string;
  active: boolean;
  interval: string;
  interval_count: number;
}

// Sample data - replace with your actual data fetching logic
const mockPlans: Plan[] = [
  {
    id: "1",
    name: "Basic Plan",
    active: true,
    interval: "month",
    interval_count: 1,
  },
  {
    id: "2",
    name: "Pro Plan",
    active: true,
    interval: "month",
    interval_count: 1,
  },
  {
    id: "3",
    name: "Enterprise Plan",
    active: false,
    interval: "year",
    interval_count: 1,
  },
  {
    id: "4",
    name: "Weekly Basic",
    active: true,
    interval: "week",
    interval_count: 1,
  },
];

// Empty state component
const EmptyState = () => (
  <div className="flex flex-col items-center justify-center py-16 px-4">
    <div className="flex items-center justify-center w-16 h-16 mb-4 bg-ui-bg-subtle rounded-full">
      <svg
        width="24"
        height="24"
        viewBox="0 0 24 24"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        className="text-ui-fg-muted"
      >
        <path
          d="M9 12L11 14L15 10M21 12C21 16.9706 16.9706 21 12 21C7.02944 21 3 16.9706 3 12C3 7.02944 7.02944 3 12 3C16.9706 3 21 7.02944 21 12Z"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      </svg>
    </div>
    <Heading className="mb-2">No pricing plans yet</Heading>
    <Text className="text-ui-fg-subtle text-center mb-6 max-w-md">
      Create your first pricing plan to start offering subscription services to
      your customers.
    </Text>
    <Button size="small">
      <PlusIcon />
      Create Plan
    </Button>
  </div>
);

const RowActions = () => (
  <ActionMenu
    groups={[
      {
        actions: [
          {
            icon: <Pencil />,
            label: "Edit",
            onClick: () => {
              alert("You clicked the edit action!");
            },
          },
        ],
      },
    ]}
  />
);

export default function PricingPlansAdmin() {
  // Replace with your actual data fetching logic
  const [plans] = useState<Plan[]>(mockPlans); // Use mockPlans or [] for empty state
  const [isLoading] = useState(false);

  // Setup column helper
  const columnHelper = createDataTableColumnHelper<Plan>();

  // Define columns
  const columns = useMemo(
    () => [
      columnHelper.accessor("name", {
        header: "Plan Name",
        enableSorting: true,
      }),
      columnHelper.accessor("active", {
        header: "Status",
        cell: ({ getValue }) => {
          const isActive = getValue();
          return (
            <Badge color={isActive ? "green" : "grey"}>
              {isActive ? "Active" : "Inactive"}
            </Badge>
          );
        },
      }),
      columnHelper.accessor("interval", {
        header: "Billing Interval",
        cell: ({ getValue, row }) => {
          const interval = getValue();
          const count = row.original.interval_count;
          const displayInterval =
            count === 1 ? interval : `${count} ${interval}s`;
          return <Text className="capitalize">{displayInterval}</Text>;
        },
      }),
      columnHelper.display({
        id: "actions",
        header: "Actions",
        cell: () => <RowActions />,
      }),
    ],
    [columnHelper],
  );

  // Configure the data table
  const table = useDataTable({
    columns,
    data: plans,
    getRowId: (plan) => plan.id,
    rowCount: plans.length,
    isLoading,
    onRowClick: (event, row) => {
      console.log("Row clicked:", row.id);
      // Handle row click - navigate to plan details, etc.
    },
  });

  // Header actions
  const headerActions = (
    <div className="flex items-center gap-2">
      <Button size="small">
        <PlusIcon />
        Create Plan
      </Button>
    </div>
  );

  return (
    <Container className="p-0">
      {/* Header */}
      <div className="flex items-center justify-between px-6 py-4 border-b border-ui-border-base">
        <div>
          <Heading level="h1">Pricing Plans</Heading>
          <Text className="text-ui-fg-subtle mt-1">
            Manage your subscription pricing plans and billing intervals
          </Text>
        </div>
        {plans.length > 0 && headerActions}
      </div>

      {/* Table or Empty State */}
      {plans.length === 0 ? (
        <EmptyState />
      ) : (
        <div className="px-6">
          <DataTable instance={table}>
            <DataTable.Toolbar className="flex items-center justify-between py-4">
              <div className="flex items-center gap-2">
                <Text className="text-ui-fg-subtle">
                  {plans.length} plan{plans.length !== 1 ? "s" : ""}
                </Text>
              </div>
            </DataTable.Toolbar>
            <DataTable.Table
              emptyState={{
                title: "No plans found",
                description:
                  "Try adjusting your search or filters to find what you're looking for.",
              }}
            />
          </DataTable>
        </div>
      )}
    </Container>
  );
}

export const config = defineRouteConfig({
  label: "Plans",
});
