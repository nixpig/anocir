create table if not exists containers_ (
  id_ varchar(1024) primary key not null,
  version_ varchar(16) not null,
  bundle_ varchar(1024) not null,
  pid_ integer not null,
  status_ varchar(16) not null,
  config_ string,
  created_at_ datetime without time zone default current_timestamp
);

create table if not exists annotations_ (
  id_ integer primary key autoincrement not null,
  container_id_ varchar(1024) not null,
  key_ varchar(1024) not null,
  value_ varchar(1024) not null,

  foreign key (container_id_) references containers_(id_)
)
