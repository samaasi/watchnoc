"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { ShieldCheck, GitBranch, FileText, ChartLine, Gear } from "@phosphor-icons/react";
import { OrganizationSwitcher, UserButton } from "@clerk/nextjs";
import { cn } from "@/lib/utils";

const nav = [
  { href: "/",         label: "Deployments", icon: GitBranch },
  { href: "/evidence", label: "Evidence",    icon: FileText },
  { href: "/dora",     label: "DORA",        icon: ChartLine },
  { href: "/settings", label: "Settings",    icon: Gear },
];

export function Sidebar() {
  const pathname = usePathname();
  return (
    <aside className="flex h-screen w-56 shrink-0 flex-col border-r border-border bg-sidebar">
      <div className="flex items-center justify-between border-b border-border px-4 py-4">
        <div className="flex items-center gap-2">
          <ShieldCheck weight="fill" className="size-5 text-primary" />
          <span className="text-sm font-semibold tracking-tight">DeployGuard</span>
        </div>
      </div>
      <div className="p-3 border-b border-border">
        <OrganizationSwitcher 
          hidePersonal
          appearance={{
            elements: {
              rootBox: "w-full",
              organizationSwitcherTrigger: "w-full flex justify-between p-2 hover:bg-sidebar-accent rounded-md text-sm",
              organizationPreviewTextContainer: "truncate",
              organizationPreviewMainIdentifier: "text-sidebar-foreground font-medium",
            }
          }}
        />
      </div>
      <nav className="flex flex-col gap-0.5 p-2 flex-1">
        {nav.map(({ href, label, icon: Icon }) => (
          <Link
            key={href}
            href={href}
            className={cn(
              "flex items-center gap-2.5 rounded px-3 py-2 text-sm transition-colors",
              pathname === href
                ? "bg-sidebar-accent text-sidebar-accent-foreground font-medium"
                : "text-sidebar-foreground hover:bg-sidebar-accent/60 hover:text-sidebar-accent-foreground"
            )}
          >
            <Icon className="size-4 shrink-0" />
            {label}
          </Link>
        ))}
      </nav>
      <div className="mt-auto border-t border-border px-4 py-3 flex items-center justify-between">
        <div className="text-xs text-muted-foreground font-medium">SOC 2 CC8.1</div>
        <UserButton appearance={{ elements: { userButtonAvatarBox: "size-6" } }} />
      </div>
    </aside>
  );
}
