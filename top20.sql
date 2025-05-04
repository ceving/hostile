copy (
  select row_to_json(top20) as top20 from (
    with data as (
      with param as (select count(*) as total from hostile)
      select
        cc as a2,
        count(*) as attacks,
        round(cast(count(*)::real * 100.0 / param.total::real as numeric), 2) as percent
      from param, hostile
      group by cc, total
      order by attacks desc
      limit 20)
    select
      row_number() over (order by attacks desc) as rank,
      name || ' (' || a2 || ')' as country,
      attacks,
      percent
    from data
    left join country using (a2)
    order by rank
  ) top20
) to stdout;
