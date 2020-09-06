from argparse import ArgumentParser
import _pickle as pickle
import os
import MySQLdb
import MySQLdb.cursors


def dbh():
    return MySQLdb.connect(
        host=os.getenv('MYSQL_HOSTNAME', 'localhost'),
        port=int(os.getenv('MYSQL_PORT', 3306)),
        user=os.getenv('MYSQL_USER', 'isutrain'),
        password=os.getenv('MYSQL_PASSWORD', 'isutrain'),
        db=os.getenv('MYSQL_DATABASE', 'isutrain'),
        charset='utf8mb4',
        cursorclass=MySQLdb.cursors.DictCursor,
        autocommit=True,
    )


def main():
    parser = ArgumentParser()
    parser.add_argument("-t", "--table_name", type=str)
    parser.add_argument("-o", "--order_by_column", type=str)
    args = parser.parse_args()

    try:
        conn = dbh()
        station_list = []
        with conn.cursor() as c:
            sql = "SELECT * FROM `{}`".format(args.table_name)
            if args.order_by_column:
                sql += 'ORDER BY `{}`'.format(args.order_by_column)

            c.execute(sql)

            while True:
                station = c.fetchone()

                if station is None:
                    break

                station_list.append(station)

    except MySQLdb.Error as err:
        app.logger.exception(err)
        raise HttpException(requests.codes['internal_server_error'], "db error")

    filename = args.table_name
    if args.order_by_column:
        filename += '_order_by_{}'.format(args.order_by_column)

    with open("{}.pkl".format(filename), "wb") as fout:
        pickle.dump(station_list, fout)


if __name__ == "__main__":
    main()
