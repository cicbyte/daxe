from pathlib import Path
import subprocess
import logging as log
import os
import shutil
import sys
import time

log.basicConfig(level=log.INFO, format="%(asctime)s - %(levelname)s - %(message)s")

# 全局配置
USE_GOX = False


def run_command(cmd, cwd=None, shell=False, direct_output=False):
    """运行命令并实时显示输出"""
    if direct_output:
        process = subprocess.Popen(
            cmd,
            stdout=sys.stdout,
            stderr=sys.stderr,
            cwd=cwd,
            shell=shell,
            universal_newlines=True
        )
        return process.wait()
    else:
        process = subprocess.Popen(
            cmd,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            encoding='utf-8',
            errors='replace',
            cwd=cwd,
            shell=shell,
            bufsize=1,
            universal_newlines=True
        )

        while True:
            output = process.stdout.readline()
            if output == '' and process.poll() is not None:
                break
            if output:
                log.info(output.strip())

        return_code = process.poll()

        stderr = process.stderr.read()
        if stderr:
            log.error(stderr.strip())

        return return_code


def check_upx():
    """检查UPX是否可用"""
    try:
        return_code = run_command(["upx", "--version"])
        return return_code == 0
    except FileNotFoundError:
        return False


def compress_with_upx(output_name):
    """使用UPX压缩可执行文件"""
    if not check_upx():
        log.info("UPX未安装，跳过压缩步骤")
        return

    try:
        original_size = os.path.getsize(output_name)
        log.info(f"开始UPX压缩，原始文件大小: {original_size/1024/1024:.2f}MB")

        return_code = run_command(
            ["upx", "--best", "--verbose", output_name],
            direct_output=True
        )

        if return_code == 0:
            compressed_size = os.path.getsize(output_name)
            compression_ratio = (1 - compressed_size / original_size) * 100
            log.info(f"压缩完成: {original_size/1024/1024:.2f}MB -> {compressed_size/1024/1024:.2f}MB (压缩率: {compression_ratio:.1f}%)")

            # 清理UPX临时文件
            base_name = os.path.splitext(output_name)[0]
            for temp_file in [f"{base_name}.000", f"{base_name}.upx"]:
                if os.path.exists(temp_file):
                    try:
                        os.remove(temp_file)
                        log.info(f"已清理临时文件: {temp_file}")
                    except Exception as e:
                        log.warning(f"清理临时文件 {temp_file} 失败: {str(e)}")
        else:
            log.error("UPX压缩失败")
    except Exception as e:
        log.error(f"UPX压缩时出错: {str(e)}")


def check_gox():
    """检查gox是否已安装"""
    try:
        return_code = run_command(["gox", "-h"])
        return return_code == 0
    except FileNotFoundError:
        return False


def get_version():
    """获取版本号
    优先使用 Git 标签作为版本号，如果没有标签则使用 Git 提交哈希
    """
    try:
        tag = subprocess.check_output(
            ["git", "describe", "--tags", "--abbrev=0"],
            stderr=subprocess.DEVNULL,
            universal_newlines=True
        ).strip()
        return tag
    except subprocess.CalledProcessError:
        try:
            commit = subprocess.check_output(
                ["git", "rev-parse", "--short", "HEAD"],
                stderr=subprocess.DEVNULL,
                universal_newlines=True
            ).strip()
            return f"0.0.0-{commit}"
        except subprocess.CalledProcessError:
            return "0.0.0-dev"


def get_build_info():
    """获取构建信息"""
    try:
        commit = subprocess.check_output(
            ["git", "rev-parse", "HEAD"],
            stderr=subprocess.DEVNULL,
            universal_newlines=True
        ).strip()

        branch = subprocess.check_output(
            ["git", "rev-parse", "--abbrev-ref", "HEAD"],
            stderr=subprocess.DEVNULL,
            universal_newlines=True
        ).strip()

        return {
            "commit": commit,
            "branch": branch,
            "build_time": time.strftime("%Y-%m-%d %H:%M:%S")
        }
    except subprocess.CalledProcessError:
        return {
            "commit": "unknown",
            "branch": "unknown",
            "build_time": time.strftime("%Y-%m-%d %H:%M:%S")
        }


def build_with_gox():
    """使用gox进行交叉编译"""
    if not check_gox():
        log.error("gox未安装，请先安装gox: go install github.com/mitchellh/gox@v1.0.1")
        raise Exception("gox未安装")

    try:
        output_dir = "build"

        if os.path.exists(output_dir):
            shutil.rmtree(output_dir)
        os.makedirs(output_dir)

        version = get_version()
        build_info = get_build_info()

        ldflags = (
            f"-X 'main.Version={version}' "
            f"-X 'main.BuildTime={build_info['build_time']}' "
            f"-X 'main.GitCommit={build_info['commit']}' "
            f"-X 'main.GitBranch={build_info['branch']}'"
        )

        return_code = run_command([
            "gox",
            "-os", "windows linux darwin",
            "-arch", "amd64",
            "-ldflags", ldflags,
            "-output", "%s/daxe_<no value>_<no value>" % output_dir
        ])

        if return_code == 0:
            log.info(f"gox交叉编译成功，版本: {version}")

            # 为Windows版本添加.exe扩展名
            for file in os.listdir(output_dir):
                if file.startswith("daxe_windows") and not file.endswith(".exe"):
                    old_path = os.path.join(output_dir, file)
                    new_path = old_path + ".exe"
                    os.rename(old_path, new_path)
                    log.info(f"重命名Windows可执行文件: {file} -> {file}.exe")

            # 只对当前平台的可执行文件进行UPX压缩
            current_os = "windows" if os.name == 'nt' else "linux" if os.name == 'posix' else "darwin"
            for file in os.listdir(output_dir):
                if file.startswith(f"daxe_{current_os}"):
                    file_path = os.path.join(output_dir, file)
                    if os.path.isfile(file_path):
                        log.info(f"开始压缩当前平台文件: {file}")
                        compress_with_upx(file_path)
                    else:
                        log.info(f"跳过非可执行文件: {file}")
                else:
                    log.info(f"跳过其他平台文件: {file}")
        else:
            log.error("gox交叉编译失败")
            raise Exception("gox交叉编译失败")
    except Exception as e:
        log.error(f"使用gox编译时出错: {str(e)}")
        raise


def build_go_app():
    """编译Go应用"""
    try:
        if USE_GOX:
            build_with_gox()
            return

        output_name = "daxe.exe" if os.name == 'nt' else "daxe"

        return_code = run_command(["go", "build", "-o", output_name])

        if return_code == 0:
            log.info("Go应用编译成功")
            compress_with_upx(output_name)
        else:
            log.error("Go应用编译失败")
            raise Exception("Go应用编译失败")
    except Exception as e:
        log.error(f"编译Go应用时出错: {str(e)}")
        raise


def main():
    build_go_app()


if __name__ == "__main__":
    main()
