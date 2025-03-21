# -*- coding: utf-8 -*-
"""
Created on Thu May 21 13:34:37 2020

对邻接钻孔土质状态影响概率进行反距离加权插值，给出查询点处土质的极大似然估计

@author: Da YIN
"""
import numpy as np
import pandas as pd


# =============================================================================
# 计算xy平面上两点距离
# =============================================================================
def get_dist(h_id, holeID, p_query):
    delt_x = holeID.iloc[h_id, 1] - p_query[0]
    delt_y = holeID.iloc[h_id, 2] - p_query[1]
    dist = np.sqrt(np.power(delt_x, 2) + np.power(delt_y, 2))

    return dist


# =============================================================================
# 获取查询点周边钻孔索引
# =============================================================================
def get_nbh(p_query, holeID):
    points = holeID.values[:, 1:3]
    points = np.append(points, p_query[0:2])  # 添加查询点坐标
    points = points.reshape(len(holeID) + 1, 2)

    from voronoi import drilling_floor_plan
    nbh = drilling_floor_plan(points)

    return nbh


# =============================================================================
# 计算反距离加权后的土质概率
# =============================================================================
def IDW(theta, p_query, holeID, nbh, df2):
    import get_geostatus as gg

    z = p_query[2]  # 查询点高程

    D = 0
    # 计算查询点与所有相邻钻孔的反距离
    for i in range(len(nbh[len(holeID)])):
        h_id = nbh[len(holeID)][i][0]  # 索引查询相邻钻孔的邻点ID
        dist = get_dist(h_id, holeID, p_query)
        D += 1 / dist
    D = 1 / D

    set_update = {}

    for i in range(len(nbh[len(holeID)])):
        h_id = nbh[len(holeID)][i][0]  # 指定钻孔索引

        a0 = gg.get_section(h_id, z + theta, df2, holeID)  # 查询区间顶面所在土层索引号
        # a1 = gg.get_section(h_id, z, df2, holeID)
        a2 = gg.get_section(h_id, z - theta, df2, holeID)  # 查询区间底面所在土层索引号

        set_geostatus = gg.get_set(h_id, a0, a2, z, theta, df2, holeID)
        # gg.check_probability(set_geostatus)
        # print(i,h_id,set_geostatus)

        dist = get_dist(h_id, holeID, p_query)

        for j in range(len(set_geostatus)):
            key = set_geostatus[j][0]
            if key in set_update:
                set_update[key] += D / dist * set_geostatus[j][1]
            else:
                set_update[key] = D / dist * set_geostatus[j][1]

    return set_update


def estimate(df, holeID, df2):
    theta = 0.5
    for j in range(df.shape[0]):
        p_query = (df.values[j, :]).astype('float')
        # p_query = coordinates[j]    # 测试点坐标
        p_query = [p_query[0], p_query[1], -p_query[2]]  # z坐标改为负数
        nbh = get_nbh(p_query, holeID)  # 钻孔和查询点的邻点，查询点的邻点在末尾
        set_update = IDW(theta, p_query, holeID, nbh, df2)

        MLE = max(zip(set_update.values(), set_update.keys()))

        if MLE[1] != "null":
            tmp = df2.iloc[:, 5]
            for i in range(len(tmp)):
                if tmp[i] == MLE[1]:
                    flag = i
                    break
            soil = df2.iloc[flag, 4]
            df.loc[j, "识别结果"] = soil
            print(df.iloc[[j]])  # 显示识别出的这一行
            # if soil == coordinates.iloc[j, 2]:
            # count = count + 1
        else:
            if len(set_update) >= 2:
                sorted_values = sorted(set_update.values(), reverse=True)  # 按值降序排序
                second_largest_value = sorted_values[1]  # 获取第二大值
                second_largest_keys = [key for key, value in set_update.items() if
                                       value == second_largest_value]  # 获取对应的键
                tmp = df2.iloc[:, 5]
                for i in range(len(tmp)):
                    if tmp[i] == second_largest_keys[0]:
                        flag = i
                        break
                soil = df2.iloc[flag, 4]
                df.loc[j, "识别结果"] = soil
                print(df.iloc[[j]])  # 显示识别出的这一行
            else:
                df.loc[j, "识别结果"] = 'null'
                print(df.iloc[[j]])  # 显示识别出的这一行

        if j == df.shape[0] - 1:
            # 设置显示的最大行数和最大列数
            pd.set_option('display.max_rows', None)
            pd.set_option('display.max_columns', None)

    return df


def state_estimation(data, mdb_path):
    import read_LZ_data as rd

    connect_str = mdb_path
    all_table_dict = rd.get_all_tables(connect_str)

    # 提取钻孔数据
    ZUANKONG_all_data = rd.select_table(connect_str, all_table_dict, "KC_ZUANKONG")
    ZUANKONG = ZUANKONG_all_data[["ZKID", "ZKZBX", "ZKZBY", "KKGC"]]
    ZUANKONG.loc[:, ["KKGC"]] = ZUANKONG[["KKGC"]].astype('float')
    holeID = ZUANKONG.rename(columns={"ZKID": "ID", "ZKZBX": "X", "ZKZBY": "Y", "KKGC": "Z"})

    # 提取地层数据
    ZUANKONG_DICENG_all_data = rd.select_table(connect_str, all_table_dict, "KC_ZUANKONG_DICENG")
    ZUANKONG_DICENG = ZUANKONG_DICENG_all_data[["ZKID", "CDSD", "DCBH", "YXMC"]]
    ZUANKONG_DICENG.columns = ["LocationID", "DepthBase", "LegendCode", "GeologyCode"]
    ZUANKONG_DICENG.insert(1, "DepthTop", rd.append_DepthTop(ZUANKONG_DICENG_all_data))
    df2 = rd.SID(ZUANKONG_DICENG)


    data = pd.DataFrame(data, columns=['x', 'y', 'z'])

    # 批量土质状态估计和估计精度计算
    data1 = estimate(data, holeID, df2)

    return data1



# =============================================================================
# TEST
# =============================================================================
if __name__ == "__main__":
    connect_str = './test1.mdb'
    file_path = 'design27m.xyz'

    points = np.loadtxt(file_path, usecols=(0, 1, 2))


    data1 = state_estimation(points, connect_str)

    print('data1')

