# -*- coding: utf-8 -*-
"""
Created on Mon July 10 11:30:52 2023

Read borehole geological data from mdb.
Generate computable data structures from the read excel data.

pandas的数据结构介绍-CSDN博客
https://blog.csdn.net/hbu_pig/article/details/80278438

Pandas DataFrame 总结 - 简书
https://www.jianshu.com/p/6e35d37e7709

This module should be fully compatible with:
    * Python >=v3.7
    * Spyder >=v4.0

"""

import numpy as np
import pypyodbc
import pandas as pd

# 读取所有表格
def get_all_tables(connect_str):
    conn = pypyodbc.win_connect_mdb(connect_str)    # 连接本地数据库
    cur = conn.cursor()                             # 创建游标
    cols_list = list(zip(*cur.columns().description))[0]    #
    # print(cols_list)
    table_info_list = cur.fetchall()
    table_df = pd.DataFrame(table_info_list)
    table_df.columns = cols_list

    # print(table_df[['table_name', 'column_name', 'type_name']])
    result = table_df.groupby(['table_name'])['column_name'].apply(list).to_dict()
    # print(type(table_df))
    # table_df.to_csv('table.csv', index=True)
    # print(result)
    return result

# 读取表格数据
def get_table_data(connect_str, table_name, columns_list, write_csv=False):
    conn = pypyodbc.win_connect_mdb(connect_str)
    cur = conn.cursor()
    sql = "select * from " + table_name
    print(sql)
    cur.execute(sql)
    # all_data = cur.fetchall()
    data = cur.fetchall()
    cur.close()
    conn.close()
    if write_csv:
        print("****************开始导出数据****************")
        # export(all_data, columns_list, table_name + "_all_data")
        export(data, columns_list, table_name + "_all_data")
    return data

# 数据输出
def export(data, columns, file_name):
    df = pd.DataFrame(data, columns=columns)
    df.to_csv('./{}.csv'.format(file_name), encoding="utf-8-sig", index=False)


# 读取指定表格数据
def select_table(connect_str, all_table_dict, table_name):
    columns_list = all_table_dict.get(table_name)
    data = get_table_data(connect_str, table_name, columns_list, False)
    table = pd.DataFrame(data, columns=columns_list)
    return table


# 增加地层数据
def append_DepthTop(ZUANKONG_DICENG_all_data):
    DepthTop = np.zeros(ZUANKONG_DICENG_all_data.shape[0], dtype=float)
    for i in range(1, ZUANKONG_DICENG_all_data.shape[0]):
        if ZUANKONG_DICENG_all_data.loc[i, "ZKID"] - ZUANKONG_DICENG_all_data.loc[i - 1, "ZKID"] == 0:
            DepthTop[i] = ZUANKONG_DICENG_all_data.loc[i - 1, "CDSD"]
        else:
            DepthTop[i] = 0
    return DepthTop



# =============================================================================
# 自动为钻孔命名和平面坐标创建对应的代号映射
# 钻探常分为前期勘探、后期补孔，不同时期钻孔的深度和土层分类的粒度会有较大差异
# 此处为了方便实现，只取LocationID为[2,33]的深孔数据进行处理
# =============================================================================
def HID(df):
    ID = [];
    X = [];
    Y = [];
    Z = [];
    for i in range(df.shape[0]):
        # if type(df[df.columns[0]][i]) is int:
        ID.append(df.iloc[i, 0])
        X.append(df.iloc[i, 1])
        Y.append(df.iloc[i, 2])
        Z.append(df.iloc[i, 3])

    data = {'ID': ID, 'X': X, 'Y': Y, 'Z': Z}
    holeID = pd.DataFrame(data)
    # 将df的前四列导入成DataFrame格式的holeID中

    return holeID


# =============================================================================
# 自动为土层命名创建对应的代号映射表（也可以考虑将常见土层的代号固定下来）
# =============================================================================
def SID(df):
    soil = [];
    tmp = "";
    for i in range(df.shape[0]):
        # 图集编号+土层命名
        #tmp = str(df.iloc[i, 3]) + " " + df.iloc[i, 4]
        tmp = df.iloc[i, 4]
        if not (tmp in soil):
            soil.append(tmp)
    # soilname中存储图集编号+土层名
    soilID = pd.DataFrame({"soilname": soil})

    ID = [];
    tmp = "";
    j = 0;
    for i in range(df.shape[0]):
        #tmp = str(df.iloc[i, 3]) + " " + df.iloc[i, 4]
        tmp =df.iloc[i, 4]
        #tmp = str(df[df.columns[3]][i]) + " " + df[df.columns[4]][i]
        # 筛选出不重复的图集编号+土层名并编码
        j = soilID.soilname[soilID.soilname.values == tmp].index.tolist()[0]
        # 将数字代号转换为字符代号"A~Z"和"a~z"
        if j < 26:
            ID.append(chr(65 + j))
        else:
            ID.append(chr(97 + j))

    #df["soilID"] = ID
    df.insert(5, "soilID", ID)

    return df


"""    
后续该处可以更新为映射，精简代码并节省内存、提升计算效率
lsit = [str1,str2,...]
tmp = pd.Series(list)
mapper = {v:k for k,v in enumerate(tmp.unique())}
as_int = tmp.map(mapper) # dtype:int64

ctmp = tmp.cat.categories
ctmp.cat.reorder_categories(mapper).cat.codes # dtype:int8
"""


# =============================================================================
# 按钻孔代号存储钻孔所涉及地质界面点的索引及标高
# =============================================================================
def TopologicalNode(df, holeID):
    tmp = "";
    hole = [];
    interface = [];
    ID = [];
    j = 0;
    k = 0;
    depth = [];
    dd = [];
    for i in range(df.shape[0]):
        if (df.iloc[i, 0] in list(holeID.ID)):
            if (df.iloc[i, 0] != tmp):
                if (ID != []):
                    ID.append(j);
                    j += 1;
                    interface.append(ID)
                    dd.append(df.iloc[k, 2])
                    depth.append(dd)

                tmp = df.iloc[i, 0]
                hole.append(tmp)
                ID = [];
                ID.append(j);
                j += 1;
                dd = [];
                dd.append(df.iloc[i, 1])

            else:
                ID.append(j);
                j += 1;
                dd.append(df.iloc[i, 1])
                k = i

    if (ID != []):
        ID.append(j);
        j += 1;
        dd.append(df.iloc[k, 2])
        interface.append(ID);
        depth.append(dd);

    interfaceID = pd.Series(interface, index=hole)
    depthID = pd.Series(depth, index=hole)

    return (interfaceID, depthID)


# =============================================================================
# 按钻孔代号存储钻孔所涉及土层的索引及土层代号
# =============================================================================
def DID(df, holeID):
    tmp = "";
    hole = [];
    ID = [];
    SD = [];
    for i in range(df.shape[0]):
        if (df.iloc[i, 0] in list(holeID.ID)):

            if (df.iloc[i, 0] != tmp):
                if (ID != []):
                    SD.append(ID)
                tmp = df.iloc[i, 0]
                hole.append(tmp)
                ID = [];
                ID.append(df.iloc[i, 5])
            else:
                ID.append(df.iloc[i, 5])

    if (ID != []):
        SD.append(ID)
    domainID = pd.Series(SD, index=hole)

    return domainID


# =============================================================================
# 从excel中读取钻孔数据
# =============================================================================
if __name__ == "__main__":
    connect_str = 'D:\\Software\\Pycharm\\projiect_11.14\\GeoStatus\\test1.mdb'
    all_table_dict = get_all_tables(connect_str)

    # 提取钻孔数据
    ZUANKONG_all_data = select_table(connect_str, all_table_dict, "KC_ZUANKONG")
    ZUANKONG = ZUANKONG_all_data[["ZKID", "ZKZBX", "ZKZBY", "KKGC"]]
    ZUANKONG[["KKGC"]] = ZUANKONG[["KKGC"]].astype('float')
    # df1 = ZUANKONG.rename(columns={"ZKID": "ID", "ZKZBX": "X", "ZKZBY": "Y", "KKGC": "Z"})
    holeID = ZUANKONG.rename(columns={"ZKID": "ID", "ZKZBX": "X", "ZKZBY": "Y", "KKGC": "Z"})
    holeID.to_csv('ZUANKONG.csv', encoding="utf-8-sig", index=False)

    # 提取地层数据
    ZUANKONG_DICENG_all_data = select_table(connect_str, all_table_dict, "KC_ZUANKONG_DICENG")
    ZUANKONG_DICENG = ZUANKONG_DICENG_all_data[["ZKID", "CDSD", "DCBH", "YXMC"]]
    ZUANKONG_DICENG.columns = ["LocationID", "DepthBase", "LegendCode", "GeologyCode"]
    ZUANKONG_DICENG.insert(1, "DepthTop", append_DepthTop(ZUANKONG_DICENG_all_data))
    ZUANKONG_DICENG.to_csv('ZUANKONG_DICENG.csv', encoding="utf-8-sig", index=False)
    df2 = ZUANKONG_DICENG

    # holeID = HID(df1)
    df2 = SID(df2)
    (interfaceID, depthID) = TopologicalNode(df2, holeID)
    domainID = DID(df2, holeID)


