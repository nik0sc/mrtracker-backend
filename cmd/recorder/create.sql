use `traintracker`;

create table recorded_position (
    id int primary key auto_increment,
    day_of_week tinyint not null,
    seconds_of_day int not null,
    time timestamp not null,
    name varchar(10) not null,
    line_repr varchar(200) not null,
    index day_second_name (day_of_week, seconds_of_day, name),
    unique index time_name (time, name)
);

create user if not exists 'traintracker_recorder'@'%'
    identified by '6d67360fa5c2fdf4f41d469ba322ac28';

grant select, insert, update, delete on traintracker.recorded_position to 'traintracker_recorder'@'%';

# insert into recorded_position (day_of_week, seconds_of_day, time, name, line_repr) values (?,?,?,?,?)