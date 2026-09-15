# Lightning Dashboard

CloudCC Lightning dashboards group report- or view-backed chart components in a visible dashboard folder. In strict MSAPI mode all dashboard writes use MetadataService plan/apply; setup-svc is used only as the behavioral reference and may be used separately for read-only runtime verification.

The aggregate contains:

- one `tp_sys_folder` row with `foldertype=lightningdashboard` when a new folder is requested;
- one `tp_sys_dashboard` root with `islightning=true`;
- zero to fifteen `tp_sys_dashboard_report` components;
- optional `tp_sys_dashboard_condition` filters for each component.

Recent dashboards are user runtime state. MetadataService does not create `tp_sys_recent_items` merely to make a newly created dashboard appear in a recent list.
